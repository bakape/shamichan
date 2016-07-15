package types

// Captcha stores captcha data for request messages that require it, if captchas
// s are enabled
type Captcha struct {
	Captcha   string `json:"captcha"`
	CaptchaID string `json:"captchaID"`
}
