package db

import (
	"database/sql"
	"meguca/common"
	"meguca/config"
	"sync"
	"time"

	"github.com/go-playground/log"
)

// Initial position of the spam score and the amount, after exceeding which, a
// captcha solution is requested.
const spamDetectionBuffer = time.Minute

var (
	spamScoreBuffer = make(map[string]time.Duration)
	spamMu          sync.RWMutex
)

// Listen for requests for clients to fill in captchas on their next post and
// periodically flush buffered spam scores to DB
func handleSpamScores() (err error) {
	go func() {
		for range time.Tick(time.Second) {
			err := func() (err error) {
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
			}()
			if err != nil {
				log.Errorf("spam score buffer flush: %s", err)
			}
		}
	}()

	captchaMsg, err := common.EncodeMessage(common.MessageCaptcha, 0)
	if err != nil {
		return
	}
	err = Listen("captcha_required", func(ip string) (err error) {
		for _, cl := range common.GetClientsByIP(ip) {
			cl.Send(captchaMsg)
		}
		return
	})
	if err != nil {
		return
	}

	spamMsg, err := common.EncodeMessage(common.MessageInvalid,
		common.ErrSpamDected)
	if err != nil {
		return
	}
	return Listen("spam_detected", func(ip string) (err error) {
		for _, cl := range common.GetClientsByIP(ip) {
			cl.Send(spamMsg)
			cl.Close(nil)
		}
		return
	})
}

// Flush spam scores from map to DB
func flushSpamScores() (err error) {
	return InTransaction(false, func(tx *sql.Tx) (err error) {
		// Prepare statements for modifying spam score
		get, err := tx.Prepare(`select score from spam_scores where ip = $1`)
		if err != nil {
			return
		}
		upsert, err := tx.Prepare(
			`insert into spam_scores (ip, score)
			values ($1, $2)
			on conflict do update
				set score = $2
				where ip = $1`)
		if err != nil {
			return
		}
		request, err := tx.Prepare(`notify captcha_required $1`)
		if err != nil {
			return
		}
		disconnect, err := tx.Prepare(`notify spam_detected $1`)
		if err != nil {
			return
		}

		var (
			now       = time.Now()
			threshold = now.Add(-spamDetectionBuffer)
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
			if score.After(now) {
				_, err = request.Exec(ip)
				if err != nil {
					return
				}
			}

			if score.Sub(now) > spamDetectionBuffer*10 {
				// This surely is not done by normal human interaction
				_, err = disconnect.Exec(ip)
				if err != nil {
					return
				}
			}
		}
		return
	})
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
	score.Add(buffered)
	return
}

// IncrementSpamScore increments spam detection score of an IP and sends
// captcha requests, if score exceeded.
// ip: IP of client
// increment: increment amount
func IncrementSpamScore(ip string, increment time.Duration) {
	if !config.Get().Captcha {
		return
	}

	spamMu.Lock()
	defer spamMu.Unlock()
	spamScoreBuffer[ip] += increment
}

// ResetSpamScore resets a spam score to zero by IP
func ResetSpamScore(ip string) (err error) {
	if !config.Get().Captcha {
		return
	}
	spamMu.Lock()
	defer spamMu.Unlock()
	delete(spamScoreBuffer, ip)
	_, err = sq.Delete("spam_scores").Where("ip = ?", ip).Exec()
	return
}

// Returns, if the user needs a captcha to proceed with usage of server
// resources
func NeedCaptcha(ip string) (bool, error) {
	conf := config.Get()
	if !conf.Captcha {
		return false, nil
	}
	spamMu.RLock()
	defer spamMu.RUnlock()

	now := time.Now()
	score, err := mergeSpamScore(spamScoreBuffer[ip],
		now.Add(-spamDetectionBuffer),
		sq.Select("score").
			From("spam_scores").
			Where("ip = ?", ip).
			QueryRow())
	return score.Before(now), err
}

// Delete spam scores that are no longer used
func expireSpamScores() error {
	_, err := sq.Delete("spam_scores").
		Where("score < ?", time.Now().Add(-spamDetectionBuffer).Unix()).
		Exec()
	return err
}
