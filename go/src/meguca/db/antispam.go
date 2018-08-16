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
		for range time.Tick(time.Second * 3) {
			err := func() (err error) {
				spamMu.Lock()
				defer spamMu.Unlock()

				if len(spamScoreBuffer) == 0 {
					return
				}
				err = flushSpamScores(spamScoreBuffer)
				clearSpamScoreBuffer()
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

// Prepare statements for modifying spam score
func prepareSpamScoreQueries(tx *sql.Tx,
) (get, upsert, request, disconnect *sql.Stmt, err error) {
	get, err = tx.Prepare(`select score from spam_scores where ip = $1`)
	if err != nil {
		return
	}
	upsert, err = tx.Prepare(
		`insert into spam_scores (ip, score)
			values ($1, $2)
			on conflict do update
				set score = $2
				where ip = $1`)
	if err != nil {
		return
	}
	request, err = tx.Prepare(`notify captcha_required $1`)
	if err != nil {
		return
	}
	disconnect, err = tx.Prepare(`notify spam_detected $1`)
	return
}

// Flush spam scores from map to DB
func flushSpamScores(scores map[string]time.Duration) (err error) {
	return InTransaction(false, func(tx *sql.Tx) (err error) {
		get, upsert, request, disconnect, err := prepareSpamScoreQueries(tx)
		if err != nil {
			return
		}
		var (
			now       = time.Now()
			threshold = now.Add(-spamDetectionBuffer)
			score     time.Time
		)
		for ip, buffered := range scores {
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
// async: buffer result and flush it to DB at a later time
// err: if async == true, err will be equal to common.ErrSpamDetected in case of
// of spam detection for this IP
func IncrementSpamScore(ip string, increment time.Duration, async bool,
) (err error) {
	if !config.Get().Captcha {
		return
	}

	spamMu.Lock()
	defer spamMu.Unlock()

	buffered, ok := spamScoreBuffer[ip]
	if async {
		spamScoreBuffer[ip] = buffered + increment
		return
	}
	if ok {
		delete(spamScoreBuffer, ip)
	}

	var isSpam bool
	err = InTransaction(false, func(tx *sql.Tx) (err error) {
		get, upsert, request, disconnect, err := prepareSpamScoreQueries(tx)
		if err != nil {
			return
		}
		now := time.Now()
		score, err := mergeSpamScore(buffered, now.Add(-spamDetectionBuffer),
			get.QueryRow(ip))
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

		isSpam = score.Sub(now) > spamDetectionBuffer*10
		if isSpam {
			_, err = disconnect.Exec(ip)
		}
		return
	})
	if err != nil {
		return
	}
	if isSpam {
		return common.ErrSpamDected
	}
	return
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

func clearSpamScoreBuffer() {
	for ip := range spamScoreBuffer {
		delete(spamScoreBuffer, ip)
	}
}

// Clears all spam detection data. Only use for tests.
func ClearSpamScores() error {
	spamMu.Lock()
	defer spamMu.Unlock()
	clearSpamScoreBuffer()
	_, err := sq.Delete("spam_scores").Exec()
	return err
}

// Returns, if the user does not trigger antispam for post creation
func CanPost(ip string) (bool, error) {
	conf := config.Get()
	if !conf.Captcha {
		return true, nil
	}
	spamMu.RLock()
	buffered := spamScoreBuffer[ip]
	spamMu.RUnlock()

	now := time.Now()
	score, err := mergeSpamScore(buffered, now.Add(-spamDetectionBuffer),
		sq.Select("score").
			From("spam_scores").
			Where("ip = ?", ip).
			QueryRow())
	threshold := now.Add(
		-time.Duration(conf.PostCreationScore) * time.Millisecond)
	return score.Before(threshold), err
}

// Delete spam scores that are no longer used
func expireSpamScores() error {
	_, err := sq.Delete("spam_scores").
		Where("score < ?", time.Now().Add(-spamDetectionBuffer).Unix()).
		Exec()
	return err
}
