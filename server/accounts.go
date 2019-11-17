package server

import "github.com/bakape/meguca/common"

var (
	errInvalidCaptcha = common.ErrAccessDenied("invalid captcha")
	errUserIDTaken    = common.ErrInvalidInput("login ID already taken")
)

// TODO: Include captcha data in all applicable these post request
