package db

import (
	"meguca/auth"
	"meguca/common"
	"meguca/config"
	"time"

	"github.com/bakape/captchouli"
)

// ValidateCaptcha with captcha backend
func ValidateCaptcha(req auth.Captcha, ip string) (err error) {
	if !config.Get().Captcha {
		return
	}
	err = captchouli.CheckCaptcha(req.CaptchaID, req.Solution)
	switch err {
	case nil:
		return
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
		if count >= 6 {
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
