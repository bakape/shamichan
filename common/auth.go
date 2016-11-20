package common

// Captcha stores captcha data for request messages that require it, if captchas
// s are enabled
type Captcha struct {
	Captcha, CaptchaID string
}
