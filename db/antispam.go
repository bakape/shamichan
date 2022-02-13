package db

import (
	"database/sql"
	"sync"
	"time"

	"github.com/bakape/captchouli/v2"
	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/go-playground/log"
)

// Initial position of the spam score and the amount, after exceeding which, a
// captcha solution is requested.
const spamDetectionThreshold = time.Minute

var (
	spamScoreBuffer = make(map[auth.Base64Token]sessionData)
	spamMu          sync.RWMutex

	// Period for how long to keep records of any captcha being solved withing
	// this period
	lastSolvedCaptchaRetention = time.Hour * 3

	// Limit of allowed incorrect captchas per hour
	incorrectCaptchaLimit = 10
)

type sessionData struct {
	score time.Duration
	ip    string
}

// Sync cache and DB spam scores
// Separated for easier testing.
func syncSpamScores() (err error) {
	spamMu.Lock()
	defer spamMu.Unlock()

	if len(spamScoreBuffer) == 0 {
		return
	}
	err = flushSpamScores()
	for session := range spamScoreBuffer {
		delete(spamScoreBuffer, session)
	}
	return
}

// Periodically flush buffered spam scores to DB
func handleSpamScores() (err error) {
	if !common.IsTest {
		go func() {
			for range time.Tick(time.Second) {
				err := syncSpamScores()
				if err != nil {
					log.Errorf("spam score buffer flush: %s", err)
				}
			}
		}()
	}
	return nil
}

// Flush spam scores from map to DB
func flushSpamScores() (err error) {
	return InTransaction(false, func(tx *sql.Tx) (err error) {
		// Prepare statements for modifying spam score
		get, err := tx.Prepare(`select score from spam_scores where token = $1`)
		if err != nil {
			return
		}
		upsert, err := tx.Prepare(
			`insert into spam_scores (token, score)
			values ($1, $2)
			on conflict (token)
			do update set score = EXCLUDED.score`)
		if err != nil {
			return
		}

		var (
			now       = time.Now().Round(time.Second)
			threshold = now.Add(-spamDetectionThreshold)
			score     time.Time
		)
		for session, data := range spamScoreBuffer {
			score, err = mergeSpamScore(
				data.score,
				threshold,
				get.QueryRow(session[:]),
			)
			if err != nil {
				return
			}
			_, err = upsert.Exec(session[:], score.Unix())
			if err != nil {
				return
			}
			if isSpam(now, score) {
				// Use latest IP to ban for spam. Makes this a bit more
				// resilient againt phones migrating towers.
				err = banForSpam(tx, data.ip)
				if err != nil {
					return
				}
			}
		}
		return
	})
}

func banForSpam(tx *sql.Tx, ip string) error {
	return systemBanTx(tx, ip, "all", "spam detected", time.Hour*48)
}

// This surely is not done by normal human interaction
func isSpam(now, score time.Time) bool {
	return score.Sub(now) > spamDetectionThreshold*10
}

// Merge buffered spam score with the one stored in the DB
func mergeSpamScore(buffered time.Duration, threshold time.Time, r rowScanner,
) (score time.Time, err error) {
	var stored int64
	err = r.Scan(&stored)
	score = time.Unix(stored, 0)
	switch err {
	case nil:
		// Keep score from descending bellow offset or initialize
		if score.Before(threshold) {
			score = threshold
		}
	case sql.ErrNoRows:
		err = nil
		score = threshold
	default:
		return
	}
	score = score.Add(buffered)
	return
}

// IncrementSpamScore increments spam detection score of a captcha session
// and sends captcha requests, if score exceeded.
//
// session: token identifying captcha session
// ip: IP of client
// increment: increment amount in milliseconds
func IncrementSpamScore(session auth.Base64Token, ip string, increment uint) {
	if !config.Get().Captcha {
		return
	}

	spamMu.Lock()
	defer spamMu.Unlock()

	spamScoreBuffer[session] = sessionData{
		score: spamScoreBuffer[session].score +
			time.Duration(increment)*time.Millisecond,
		ip: ip,
	}
}

// resetSpamScore resets a spam score to zero for captcha session
func resetSpamScore(tx *sql.Tx, session auth.Base64Token) (err error) {
	spamMu.Lock()
	defer spamMu.Unlock()

	delete(spamScoreBuffer, session)
	_, err = sq.
		Delete("spam_scores").
		Where("token = ?", session[:]).
		RunWith(tx).
		Exec()
	return
}

// NeedCaptcha returns, if the user needs a captcha
// to proceed with usage of server resources
func NeedCaptcha(session auth.Base64Token, ip string) (need bool, err error) {
	if !config.Get().Captcha {
		return
	}

	// Require a captcha, if none have been solved in 3 hours
	has, err := SolvedCaptchaRecently(session, lastSolvedCaptchaRetention)
	if err != nil {
		return
	}
	if !has {
		need = true
		return
	}

	score, err := getSpamScore(session, ip)
	if err != nil {
		return
	}
	return score.After(time.Now()), err
}

// Merge cached and DB value and return current score
func getSpamScore(
	session auth.Base64Token,
	ip string,
) (
	score time.Time,
	err error,
) {
	spamMu.RLock()
	defer spamMu.RUnlock()

	now := time.Now().Round(time.Second)
	score, err = mergeSpamScore(
		spamScoreBuffer[session].score,
		now.Add(-spamDetectionThreshold),
		sq.
			Select("score").
			From("spam_scores").
			Where("token = ?", session[:]).
			QueryRow(),
	)
	if err != nil {
		return
	}
	if isSpam(now, score) {
		err = InTransaction(false, func(tx *sql.Tx) error {
			return banForSpam(tx, ip)
		})
		if err != nil {
			return
		}
		err = common.ErrSpamDected
	}
	return
}

// Check if IP is spammer
func AssertNotSpammer(session auth.Base64Token, ip string) (err error) {
	_, err = getSpamScore(session, ip)
	return
}

// Delete spam scores that are no longer used
func expireSpamScores() error {
	_, err := sq.Delete("spam_scores").
		Where("score < ?", time.Now().Add(-spamDetectionThreshold).Unix()).
		Exec()
	return err
}

// ValidateCaptcha with captcha backend
func ValidateCaptcha(
	req auth.Captcha,
	session auth.Base64Token,
	ip string,
) (err error) {
	if !config.Get().Captcha {
		return
	}
	err = captchouli.CheckCaptcha(req.CaptchaID, req.Solution)
	switch err {
	case nil:
		return InTransaction(false, func(tx *sql.Tx) (err error) {
			_, err = sq.Insert("last_solved_captchas").
				Columns("token", "ip").
				Values(session[:], ip).
				Suffix(
					`on conflict (token) do
					update set time = now()`,
				).
				Exec()
			if err != nil {
				return
			}
			return resetSpamScore(tx, session)
		})
	case captchouli.ErrInvalidSolution:
		banned := false
		err = InTransaction(false, func(tx *sql.Tx) (err error) {
			_, err = sq.Insert("failed_captchas").
				Columns("ip", "expires").
				Values(ip, time.Now().Add(time.Hour)).
				RunWith(tx).
				Exec()
			if err != nil {
				return
			}

			var count int
			err = sq.Select("count(*)").
				From("failed_captchas").
				Where("ip = ? and expires > now() at time zone 'utc'", ip).
				RunWith(tx).
				QueryRow().
				Scan(&count)
			if err != nil {
				return
			}
			if count >= incorrectCaptchaLimit {
				err = systemBanTx(tx, ip, "all", "bot detected", time.Hour*48)
				if err != nil {
					return
				}
				banned = true
			}

			return
		})
		if err != nil {
			return
		}
		if banned {
			return common.ErrBanned
		}
		return common.ErrInvalidCaptcha
	default:
		return
	}
}

// Returns, if IP has solved a captcha within the last dur
func SolvedCaptchaRecently(
	session auth.Base64Token,
	dur time.Duration,
) (
	has bool,
	err error,
) {
	if !config.Get().Captcha {
		has = true
		return
	}
	err = sq.Select("true").
		From("last_solved_captchas").
		Where("token = ? and time > ?", session[:], time.Now().Add(-dur)).
		QueryRow().
		Scan(&has)
	if err == sql.ErrNoRows {
		err = nil
	}
	return
}

func expireLastSolvedCaptchas() (err error) {
	_, err = sq.Delete("last_solved_captchas").
		Where("time < ?", time.Now().Add(-lastSolvedCaptchaRetention)).
		Exec()
	return
}
