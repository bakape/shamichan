// Wecbsocket message handlers central file

package websockets

import (
	"encoding/json"
	"errors"
)

var (
	isTest bool

	errInValidCaptcha = errors.New("invalid captcha provided")
)

// Decode message JSON into the supplied type. Will augment, once we switch to
// a binary message protocol.
func decodeMessage(data []byte, dest interface{}) error {
	return json.Unmarshal(data, dest)
}
