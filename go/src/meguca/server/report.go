package server

import (
	"meguca/auth"
	"meguca/common"
	"meguca/db"
	"meguca/templates"
	"net/http"
	"strconv"
)

// Report a post for rule violations
func report(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, jsonLimit)
	err := r.ParseMultipartForm(0)
	if err != nil {
		text400(w, err)
		return
	}
	f := r.Form

	ip, err := auth.GetIP(r)
	if err != nil {
		text400(w, err)
		return
	}
	if !auth.AuthenticateCaptcha(auth.Captcha{
		CaptchaID: f.Get("captchaID"),
		Solution:  f.Get("captcha"),
	}, ip) {
		text403(w, errInvalidCaptcha)
		return
	}

	target, err := strconv.ParseUint(f.Get("target"), 10, 64)
	if err != nil {
		text400(w, err)
		return
	}

	board, err := db.GetPostBoard(target)
	if err != nil {
		text400(w, err)
		return
	}
	if !auth.IsNonMetaBoard(board) {
		text400(w, errInvalidBoardName)
		return
	}
	if !assertNotBanned(w, r, board) {
		return
	}

	reason := f.Get("reason")
	if len(reason) > common.MaxLenReason {
		text400(w, errReasonTooLong)
		return
	}

	err = db.Report(target, board, reason, ip, f.Get("illegal") == "on")
	if err != nil {
		text500(w, r, err)
		return
	}
}

// Render post reporting form
func reportForm(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(extractParam(r, "id"), 10, 64)
	if err != nil {
		text400(w, err)
		return
	}
	serveHTML(w, r, "", []byte(templates.ReportForm(id)), nil)
}

// Render a list of reports for the board
func reportList(w http.ResponseWriter, r *http.Request) {
	board := extractParam(r, "board")
	if !auth.IsNonMetaBoard(board) {
		text404(w)
		return
	}

	rep, err := db.GetReports(board)
	if err != nil {
		text500(w, r, err)
		return
	}
	serveHTML(w, r, "", []byte(templates.ReportList(rep)), nil)
}
