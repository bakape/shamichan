package server

// // Authenticate a captcha solution
// func authenticateCaptcha(w http.ResponseWriter, r *http.Request) {
// 	err := func() (err error) {
// 		if !assertNotBanned(w, r, "all") {
// 			return
// 		}
// 		err = r.ParseForm()
// 		if err != nil {
// 			return common.StatusError{
// 				Err:  err,
// 				Code: 400,
// 			}
// 		}

// 		ip, err := auth.GetIP(r)
// 		if err != nil {
// 			return
// 		}

// 		var (
// 			c       auth.Captcha
// 			session auth.AuthKey
// 		)
// 		c.FromRequest(r)
// 		err = session.EnsureCookie(w, r)
// 		if err != nil {
// 			return
// 		}
// 		err = db.ValidateCaptcha(c, session, ip)
// 		if err == common.ErrInvalidCaptcha {
// 			b := extractParam(r, "board")
// 			s := auth.CaptchaService(b)
// 			if s == nil {
// 				return errCaptchasNotReady(b)
// 			}
// 			return s.ServeNewCaptcha(w, r)
// 		}
// 		if err != nil {
// 			return
// 		}

// 		w.Write([]byte("OK"))
// 		return
// 	}()
// 	if err != nil {
// 		httpError(w, r, err)
// 	}
// }

// // Create new captcha and write its HTML to w.
// // Colour and background can be left blank to use defaults.
// func serveNewCaptcha(w http.ResponseWriter, r *http.Request) {
// 	httpError(w, r, func() (err error) {
// 		if !assertNotBanned(w, r, "all") {
// 			return
// 		}

// 		ip, err := auth.GetIP(r)
// 		if err != nil {
// 			return
// 		}
// 		var session auth.AuthKey
// 		err = session.EnsureCookie(w, r)
// 		if err != nil {
// 			return
// 		}
// 		db.IncrementSpamScore(session, ip, config.Get().ImageScore)

// 		s := auth.CaptchaService(b)
// 		if s == nil {
// 			return errCaptchasNotReady(b)
// 		}
// 		s.ServeNewCaptcha(w, r)
// 		return
// 	}())
// }
