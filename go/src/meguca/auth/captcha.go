package auth

// Captcha contains the ID and solution of a captcha-protected request
type Captcha struct {
	CaptchaID, Solution string
}
