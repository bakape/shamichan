//go:generate ffjson --nodecoder --force-regenerate $GOFILE

package common

import "errors"

// Common error values
var (
	ErrInvalidCreds = errors.New("invalid login credentials")
)

// Captcha stores captcha data for request messages that require it, if captchas
// s are enabled
type Captcha struct {
	Captcha, CaptchaID string
}
