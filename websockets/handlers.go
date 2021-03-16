// Websocket message handlers central file

package websockets

import (
	"encoding/json"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/websockets/feeds"
)

// Decode message JSON into the supplied type. Will augment, once we switch to
// a binary message protocol.
func decodeMessage(data []byte, dest interface{}) error {
	return json.Unmarshal(data, dest)
}

// Run the appropriate handler for the websocket message
func (c *Client) runHandler(typ common.MessageType, msg []byte) (err error) {
	data := msg[2:]
	switch typ {
	case common.MessageSynchronise:
		return c.synchronise(data)
	case common.MessageReclaim:
		return c.reclaimPost(data)
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
	case common.MessageMeguTV:
		return feeds.SubscribeToMeguTV(c)
	case common.MessagePM:
		// Check to match schema
		var req struct {
			From uint64 `json:"from"`
			To   uint64 `json:"to"`
			Text string `json:"text"`
		}
		err = decodeMessage(data, &req)
		if err != nil {
			return
		}
		var buf []byte
		buf, err = common.EncodeMessage(common.MessagePM, req)
		if err != nil {
			return
		}
		for _, c := range feeds.All() {
			c.Send(buf)
		}
		return
	default:
		return errInvalidPayload(msg)
	}
}
