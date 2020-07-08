package db

import (
	"context"
	"sync"
	"time"

	"github.com/bakape/captchouli/v2"
	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/jackc/pgx/v4"
)

// TODO: move this to Rust as images now don't check the spam score

const (
	// Amount of score, after exceeding which, a captcha solution is requested
	spamDetectionThreshold = time.Minute
)

var (
	spamScoreBuffer = make(map[uint64]time.Duration)
	spamMu          sync.RWMutex
)

// Sync cache and DB spam scores
func syncSpamScores() (err error) {
	spamMu.Lock()
	defer spamMu.Unlock()

	if len(spamScoreBuffer) == 0 {
		return
	}
	err = flushSpamScores()
	for pubKey := range spamScoreBuffer {
		delete(spamScoreBuffer, pubKey)
	}
	return
}

// Flush spam scores from buffer to DB
func flushSpamScores() (err error) {
	return InTransaction(context.Background(), func(tx pgx.Tx) (err error) {
		for pubKey, buffered := range spamScoreBuffer {
			_, err = tx.Exec(
				context.Background(),
				`insert into spam_scores as s (public_key, expires)
				values ($1, now() + $2)
				on conflict (public_key)
				do update set expires = (
					(
						case
							when s.expires < now() then now()
							else s.expires
						end
					)
					+ $2
				)`,
				pubKey,
				buffered,
			)
			if err != nil {
				return
			}
		}
		return
	})
}

// Increments spam detection score of a public key and sends captcha requests,
// if score exceeded.
//
// pubKey: private ID of user publick key
// increment: increment amount in milliseconds
func IncrementSpamScore(pubKey uint64, increment uint64) {
	if !config.Get().Captcha {
		return
	}

	spamMu.Lock()
	spamScoreBuffer[pubKey] += time.Duration(increment) * time.Millisecond
	spamMu.Unlock()
}

// NeedCaptcha returns, if the pubKey needs a captcha to proceed with usage
// of server resources
func NeedCaptcha(ctx context.Context, pubKey uint64) (need bool, err error) {
	if !config.Get().Captcha {
		return
	}

	// Require a captcha, if none have been solved in 3 hours
	has, err := SolvedCaptchaRecently(ctx, pubKey)
	if err != nil {
		return
	}
	if !has {
		need = true
		return
	}

	score, err := getSpamScore(ctx, pubKey)
	if err != nil {
		return
	}
	return score.After(time.Now().Add(spamDetectionThreshold)), err
}

// Merge cached and DB value and return current score
func getSpamScore(ctx context.Context, pubKey uint64) (
	score time.Time,
	err error,
) {
	spamMu.RLock()
	defer spamMu.RUnlock()

	now := time.Now()
	err = db.
		QueryRow(
			ctx,
			`select expires
			from spam_scores
			where public_key = $1 and expires > now()`,
			pubKey,
		).
		Scan(&score)
	switch err {
	case nil:
	case pgx.ErrNoRows:
		score = now
		err = nil
	default:
		return
	}

	score.Add(spamScoreBuffer[pubKey])

	return
}

// Check if pubKey is spammer
func AssertNotSpammer(ctx context.Context, pubKey uint64) (err error) {
	_, err = getSpamScore(ctx, pubKey)
	return
}

// Separated for unit tests
func recordValidCaptcha(ctx context.Context, pubKey uint64) (err error) {
	spamMu.Lock()
	defer spamMu.Unlock()

	delete(spamScoreBuffer, pubKey)

	return InTransaction(ctx, func(tx pgx.Tx) (err error) {
		_, err = tx.Exec(
			ctx,
			`insert into last_solved_captchas (public_key, expires)
			values ($1, now() + interval '3 hours')
			on conflict (public_key)
			do update set expires = excluded.expires`,
			pubKey,
		)
		if err != nil {
			return
		}
		_, err = tx.Exec(
			ctx,
			`delete from spam_scores
			where public_key = $1`,
			pubKey,
		)
		return
	})
}

// ValidateCaptcha with captcha backend
func ValidateCaptcha(
	ctx context.Context,
	req auth.Captcha,
	pubKey uint64,
) (err error) {
	if !config.Get().Captcha {
		return
	}

	err = captchouli.CheckCaptcha(req.CaptchaID, req.Solution)
	switch err {
	case nil:
		return recordValidCaptcha(ctx, pubKey)
	case captchouli.ErrInvalidSolution:
		return common.ErrInvalidCaptcha
	default:
		return
	}
}

// Returns, if pubKey has solved a captcha within the last 3 hours
func SolvedCaptchaRecently(ctx context.Context, pubKey uint64) (
	has bool,
	err error,
) {
	if !config.Get().Captcha {
		has = true
		return
	}

	err = db.
		QueryRow(
			ctx,
			`select exists (
				select
				from last_solved_captchas
				where public_key = $1 and expires > now()
			)`,
			pubKey,
		).
		Scan(&has)
	return
}
