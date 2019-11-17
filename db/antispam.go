package db

import (
	"net"
	"sync"
	"time"

	"github.com/bakape/captchouli"
	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/jackc/pgx"
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
	ip    net.IP
}

// Sync cache and DB spam scores
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

// Flush spam scores from map to DB
func flushSpamScores() (err error) {
	return InTransaction(func(tx *pgx.Tx) (err error) {
		var (
			now       = time.Now().Round(time.Second)
			threshold = now.Add(-spamDetectionThreshold)
			score     time.Time
		)
		for session, data := range spamScoreBuffer {
			score, err = mergeSpamScore(
				data.score,
				threshold,
				tx.QueryRow(
					"select score from spam_scores where token = $1",
					session[:],
				),
			)
			if err != nil {
				return
			}
			_, err = tx.Exec(
				`insert into spam_scores (token, score)
				values ($1, $2)
				on conflict (token)
				do update set score = EXCLUDED.score`,
				session[:],
				score.Unix(),
			)
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

func banForSpam(tx *pgx.Tx, ip net.IP) error {
	// TODO
	// return systemBanTx(tx, ip, "spam detected", time.Hour*48)
	return nil
}

// This surely is not done by normal human interaction
func isSpam(now, score time.Time) bool {
	return score.Sub(now) > spamDetectionThreshold*10
}

// Merge buffered spam score with the one stored in the DB
func mergeSpamScore(
	buffered time.Duration,
	threshold time.Time,
	r *pgx.Row,
) (
	score time.Time,
	err error,
) {
	var stored int64
	err = r.Scan(&stored)
	score = time.Unix(stored, 0)
	switch err {
	case nil:
		// Keep score from descending bellow offset or initialize
		if score.Before(threshold) {
			score = threshold
		}
	case pgx.ErrNoRows:
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
func IncrementSpamScore(session auth.Base64Token, ip net.IP, increment uint) {
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

// NeedCaptcha returns, if the user needs a captcha
// to proceed with usage of server resources
func NeedCaptcha(session auth.Base64Token, ip net.IP) (need bool, err error) {
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
	ip net.IP,
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
		db.QueryRow(
			"select score from spam_scores where token = $1",
			session[:],
		),
	)
	if err != nil {
		return
	}
	if isSpam(now, score) {
		err = InTransaction(func(tx *pgx.Tx) error {
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
func AssertNotSpammer(session auth.Base64Token, ip net.IP) (err error) {
	_, err = getSpamScore(session, ip)
	return
}

// Separated for unit tests
func recordValidCaptcha(session auth.Base64Token) (err error) {
	spamMu.Lock()
	defer spamMu.Unlock()

	delete(spamScoreBuffer, session)
	_, err = db.Exec("select validate_captcha($1::bytea)", session[:])
	return
}

// ValidateCaptcha with captcha backend
func ValidateCaptcha(
	req auth.Captcha,
	session auth.Base64Token,
	ip net.IP,
) (err error) {
	if !config.Get().Captcha {
		return
	}
	err = captchouli.CheckCaptcha(req.CaptchaID, req.Solution)
	switch err {
	case nil:
		return recordValidCaptcha(session)
	case captchouli.ErrInvalidSolution:
		var count int
		err = db.
			QueryRow(
				"select record_invalid_captcha($1::inet)",
				ip,
			).
			Scan(&count)
		if err != nil {
			return
		}
		if count >= incorrectCaptchaLimit {
			// TODO
			// err = SystemBan(ip, "bot detected", time.Hour*48)
			// if err != nil {
			// 	return
			// }
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
	err = db.
		QueryRow(
			`select exists (
				select
				from last_solved_captchas
				where token = $1 and time > $2
			)`,
			session[:],
			time.Now().Add(-dur),
		).
		Scan(&has)
	return
}
