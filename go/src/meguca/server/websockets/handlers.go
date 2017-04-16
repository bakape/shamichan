// Wecbsocket message handlers central file

package websockets

import (
	"encoding/json"
	"errors"
	"meguca/common"
)

var errInValidCaptcha = errors.New("invalid captcha provided")

// Decode message JSON into the supplied type. Will augment, once we switch to
// a binary message protocol.
func decodeMessage(data []byte, dest interface{}) error {
	return json.Unmarshal(data, dest)
}

// Run the appropriate handler for the websocket message
func (c *Client) runHandler(typ common.MessageType, msg []byte) error {
	data := msg[2:]
	switch typ {
	case common.MessageSynchronise:
		return c.synchronise(data)
	case common.MessageReclaim:
		return c.reclaimPost(data)
	case common.MessageInsertThread:
		return c.insertThread(data)
	case common.MessageAppend:
		return c.appendRune(data)
	case common.MessageBackspace:
		return c.backspace()
	case common.MessageClosePost:
		return c.closePost()
	case common.MessageSplice:
		return c.spliceText(data)
	case common.MessageInsertPost:
		return c.insertPost(data)
	case common.MessageInsertImage:
		return c.insertImage(data)
	case common.MessageNOOP:
		// No operation message handler. Used as a one way pseudo-ping.
		return nil
	case common.MessageSpoiler:
		return c.spoilerImage()
	default:
		return errInvalidPayload(msg)
	}
}
