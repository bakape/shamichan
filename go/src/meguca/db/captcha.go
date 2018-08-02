package db

import (
	"database/sql"
	"meguca/auth"
	"meguca/common"
	"meguca/config"
	"net/http"
	"time"

	"github.com/dchest/captcha"
	"github.com/go-playground/log"
)

const captchaLifetime = time.Minute * 20

var captchaServer = captcha.Server(captcha.StdWidth, captcha.StdHeight)

// Implements captcha.Store
type dbCaptchaStore struct{}

// TODO: Properly propagate errors, when we switch to the in-house captcha
// system
func (dbCaptchaStore) Set(id string, digits []byte) {
	// Don't know what the package does with the array later, so bet to copy
	buf := make([]byte, len(digits))
	for i := range digits {
		buf[i] = digits[i] + '0'
	}

	// TODO: This can actually fail on ID collision. Need to account for this,
	// when we migrate to an in-house captcha system.
	_, err := sq.Insert("captchas").
		Columns("id", "solution", "expires").
		Values(id, string(buf), time.Now().Add(captchaLifetime).UTC()).
		Exec()
	if err != nil {
		log.Errorf("captcha: %s", err)
	}
}

func (dbCaptchaStore) Get(id string, clear bool) (digits []byte) {
	solution, err := getCaptcha(id, clear)
	if err != nil && err != sql.ErrNoRows {
		log.Errorf("captcha: %s", err)
	}

	digits = []byte(solution)
	for i := range digits {
		digits[i] = digits[i] - '0'
	}
	return
}

// Get captcha from DB and optionally remove it afterwards
func getCaptcha(id string, clear bool) (solution string, err error) {
	err = InTransaction(false, func(tx *sql.Tx) (err error) {
		r, err := withTransaction(tx, sq.
			Select("solution").
			From("captchas").
			Where("id = ? and expires > now() at time zone 'utc'", id)).
			QueryRow()
		if err != nil {
			return
		}
		err = r.Scan(&solution)
		if err != nil {
			return
		}

		if clear {
			err = withTransaction(tx, sq.
				Delete("captchas").
				Where("id = ?", id)).
				Exec()
		}
		return
	})
	return
}

func initCaptchas() error {
	captcha.SetCustomStore(new(dbCaptchaStore))
	return nil
}

// NewCaptchaID creates a new captcha and write its ID to the client
func NewCaptchaID(w http.ResponseWriter, _ *http.Request) {
	h := w.Header()
	h.Set("Content-Type", "text/plain")
	h.Set("Cache-Control", "no-store, private")
	w.Write([]byte(captcha.New()))
}

// ServeCaptcha serves captcha images and audio
func ServeCaptcha(w http.ResponseWriter, r *http.Request) {
	captchaServer.ServeHTTP(w, r)
}

// AuthenticateCaptcha with captcha backend
func AuthenticateCaptcha(req auth.Captcha, ip string) (err error) {
	if !config.Get().Captcha {
		return
	}
	ok := captcha.VerifyString(req.CaptchaID, req.Solution)
	if ok {
		return
	}

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
	if count >= 3 {
		err = SystemBan(ip, "bot detected", time.Now().Add(time.Hour*48))
		if err != nil {
			return
		}
	}

	return common.ErrInvalidCaptcha
}
