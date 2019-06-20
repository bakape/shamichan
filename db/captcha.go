package db

import (
	"database/sql"
	"time"

	"github.com/bakape/captchouli"
	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
)

const (
	// Period for how long to keep records of any captcha being solved withing
	// this period
	lastSolvedCaptchaRetention = time.Hour * 3

	// Limit of allowed incorrect captchas per hour
	incorrectCaptchaLimit = 10
)

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
		_, err = sq.Insert("last_solved_captchas").
			Columns("token").
			Values(session).
			Suffix(
				`on conflict (ip) do
				update set time = now() at time zone 'utc'`).
			Exec()
		if err != nil {
			return
		}
		return resetSpamScore(ip)
	case captchouli.ErrInvalidSolution:
		_, err = sq.Insert("failed_captchas").
			Columns("ip", "expires").
			Values(ip, time.Now().Add(time.Hour).UTC()).
			Exec()
		if err != nil {
			return
		}

		var count int
		err = sq.Select("count(*)").
			From("failed_captchas").
			Where("ip = ? and expires > now() at time zone 'utc'", ip).
			QueryRow().
			Scan(&count)
		if err != nil {
			return
		}
		if count >= incorrectCaptchaLimit {
			err = SystemBan(ip, "bot detected", time.Hour*48)
			if err != nil {
				return
			}
			return common.ErrBanned
		}
		return common.ErrInvalidCaptcha
	default:
		return
	}
}

// Returns, if IP has solved a captcha within the last dur
func SolvedCaptchaRecently(ip string, dur time.Duration) (has bool, err error) {
	if !config.Get().Captcha {
		has = true
		return
	}
	err = sq.Select("true").
		From("last_solved_captchas").
		Where("ip = ? and time > ?", ip, time.Now().UTC().Add(-dur)).
		QueryRow().
		Scan(&has)
	if err == sql.ErrNoRows {
		err = nil
	}
	return
}

func expireLastSolvedCaptchas() (err error) {
	_, err = sq.Delete("last_solved_captchas").
		Where("time < ?", time.Now().UTC().Add(-lastSolvedCaptchaRetention)).
		Exec()
	return
}
