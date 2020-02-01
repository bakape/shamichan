package db

import (
	"context"
	"sync"
	"time"

	"github.com/bakape/captchouli"
	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/jackc/pgx/v4"
)

const (
	// Amount of score, after exceeding which, a captcha solution is requested
	spamDetectionThreshold = time.Minute
)

var (
	spamScoreBuffer = make(map[auth.AuthKey]time.Duration)
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
	for user := range spamScoreBuffer {
		delete(spamScoreBuffer, user)
	}
	return
}

// Flush spam scores from buffer to DB
func flushSpamScores() (err error) {
	return InTransaction(context.Background(), func(tx pgx.Tx) (err error) {
		for user, buffered := range spamScoreBuffer {
			_, err = tx.Exec(
				context.Background(),
				`insert into spam_scores as s (auth_key, expires)
				values ($1, now() + $2)
				on conflict (auth_key)
				do update set expires = (
					(
						case
							when s.expires < now() then now()
							else s.expires
						end
					)
					+ $2
				)`,
				user,
				buffered,
			)
			if err != nil {
				return
			}
		}
		return
	})
}

// IncrementSpamScore increments spam detection score of a captcha session
// and sends captcha requests, if score exceeded.
//
// user: token identifying user
// increment: increment amount in milliseconds
func IncrementSpamScore(user auth.AuthKey, increment uint) {
	if !config.Get().Captcha {
		return
	}

	spamMu.Lock()
	spamScoreBuffer[user] += time.Duration(increment) * time.Millisecond
	spamMu.Unlock()
}

// NeedCaptcha returns, if the user needs a captcha to proceed with usage
// of server resources
func NeedCaptcha(
	ctx context.Context,
	user auth.AuthKey,
) (need bool, err error) {
	if !config.Get().Captcha {
		return
	}

	// TODO: Check, if globally banned

	// Require a captcha, if none have been solved in 3 hours
	has, err := SolvedCaptchaRecently(ctx, user)
	if err != nil {
		return
	}
	if !has {
		need = true
		return
	}

	score, err := getSpamScore(ctx, user)
	if err != nil {
		return
	}
	return score.After(time.Now().Add(spamDetectionThreshold)), err
}

// Merge cached and DB value and return current score
func getSpamScore(ctx context.Context, user auth.AuthKey) (
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
			where auth_key = $1 and expires > now()`,
			user,
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

	score.Add(spamScoreBuffer[user])

	return
}

// Check if user is spammer
func AssertNotSpammer(ctx context.Context, user auth.AuthKey) (err error) {
	_, err = getSpamScore(ctx, user)
	return
}

// Separated for unit tests
func recordValidCaptcha(ctx context.Context, user auth.AuthKey) (err error) {
	spamMu.Lock()
	defer spamMu.Unlock()

	delete(spamScoreBuffer, user)

	return InTransaction(ctx, func(tx pgx.Tx) (err error) {
		_, err = tx.Exec(
			ctx,
			`insert into last_solved_captchas (auth_key, expires)
			values ($1, now() + interval '3 hours')
			on conflict (auth_key)
			do update set expires = excluded.expires`,
			user,
		)
		if err != nil {
			return
		}
		_, err = tx.Exec(
			ctx,
			`delete from spam_scores
			where auth_key = $1`,
			user,
		)
		return
	})
}

// ValidateCaptcha with captcha backend
func ValidateCaptcha(
	ctx context.Context,
	req auth.Captcha,
	user auth.AuthKey,
) (err error) {
	if !config.Get().Captcha {
		return
	}

	err = captchouli.CheckCaptcha(req.CaptchaID, req.Solution)
	switch err {
	case nil:
		return recordValidCaptcha(ctx, user)
	case captchouli.ErrInvalidSolution:
		return common.ErrInvalidCaptcha
	default:
		return
	}
}

// Returns, if user has solved a captcha within the last 3 hours
func SolvedCaptchaRecently(ctx context.Context, user auth.AuthKey) (
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
				where auth_key = $1 and expires > now()
			)`,
			user,
		).
		Scan(&has)
	return
}
