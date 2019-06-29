package db

import (
	"database/sql"
	"sync"
	"time"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/go-playground/log"
)

// Initial position of the spam score and the amount, after exceeding which, a
// captcha solution is requested.
const spamDetectionThreshold = time.Minute

var (
	spamScoreBuffer = make(map[string]time.Duration)
	spamMu          sync.RWMutex
)

// Sync cache and DB spam scores
// Separated for testing.
func syncSpamScores() (err error) {
	spamMu.Lock()
	defer spamMu.Unlock()

	if len(spamScoreBuffer) == 0 {
		return
	}
	err = flushSpamScores()
	for ip := range spamScoreBuffer {
		delete(spamScoreBuffer, ip)
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
	return InTransaction(func(tx *sql.Tx) (err error) {
		// Prepare statements for modifying spam score
		get, err := tx.Prepare(`select score from spam_scores where ip = $1`)
		if err != nil {
			return
		}
		upsert, err := tx.Prepare(
			`insert into spam_scores (ip, score)
			values ($1, $2)
			on conflict (ip) do update
				set score = EXCLUDED.score`)
		if err != nil {
			return
		}

		var (
			now       = time.Now().Round(time.Second)
			threshold = now.Add(-spamDetectionThreshold)
			score     time.Time
		)
		for ip, buffered := range spamScoreBuffer {
			score, err = mergeSpamScore(buffered, threshold, get.QueryRow(ip))
			if err != nil {
				return
			}
			_, err = upsert.Exec(ip, score.Unix())
			if err != nil {
				return
			}
			if isSpam(now, score) {
				err = banForSpam(tx, ip)
				if err != nil {
					return
				}
			}
		}
		return
	})
}

func banForSpam(tx *sql.Tx, ip string) error {
	return systemBanTx(tx, ip, "spam detected", time.Hour*48)
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

// IncrementSpamScore increments spam detection score of an IP and sends
// captcha requests, if score exceeded.
// ip: IP of client
// increment: increment amount in milliseconds
func IncrementSpamScore(ip string, increment uint) {
	if !config.Get().Captcha {
		return
	}

	spamMu.Lock()
	defer spamMu.Unlock()
	spamScoreBuffer[ip] += time.Duration(increment) * time.Millisecond
}

// resetSpamScore resets a spam score to zero by IP
func resetSpamScore(ip string) (err error) {
	if !config.Get().Captcha {
		return
	}
	spamMu.Lock()
	defer spamMu.Unlock()
	delete(spamScoreBuffer, ip)
	_, err = sq.Delete("spam_scores").Where("ip = ?", ip).Exec()
	return
}

// NeedCaptcha returns, if the user needs a captcha
// to proceed with usage of server resources
func NeedCaptcha(ip string) (need bool, err error) {
	if !config.Get().Captcha {
		return
	}

	// Require a captcha, if none have been solved in 3 hours
	has, err := SolvedCaptchaRecently(ip, lastSolvedCaptchaRetention)
	if err != nil {
		return
	}
	if !has {
		need = true
		return
	}

	score, err := getSpamScore(ip)
	if err != nil {
		return
	}
	return score.After(time.Now()), err
}

// Merge cached and DB value and return current score
func getSpamScore(ip string) (score time.Time, err error) {
	spamMu.RLock()
	defer spamMu.RUnlock()

	now := time.Now().Round(time.Second)
	score, err = mergeSpamScore(spamScoreBuffer[ip],
		now.Add(-spamDetectionThreshold),
		sq.Select("score").
			From("spam_scores").
			Where("ip = ?", ip).
			QueryRow())
	if err != nil {
		return
	}
	if isSpam(now, score) {
		err = InTransaction(func(tx *sql.Tx) error {
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
func AssertNotSpammer(ip string) (err error) {
	_, err = getSpamScore(ip)
	return
}

// Delete spam scores that are no longer used
func expireSpamScores() error {
	_, err := sq.Delete("spam_scores").
		Where("score < ?", time.Now().Add(-spamDetectionThreshold).Unix()).
		Exec()
	return err
}
