// Wecbsocket message handlers central file

package websockets

import (
	"encoding/json"
	"errors"
	"strconv"
)

// MessageType is the identifier code for websocket message types
type MessageType uint8

// 1 - 29 modify post model state
const (
	MessageInvalid MessageType = iota
	MessageInsertThread
	MessageInsertPost
	MessageAppend
	MessageBackspace
	MessageSplice
	MessageClosePost
	MessageLink
	MessageBacklink
	MessageCommand
	MessageInsertImage
	MessageSpoiler
)

// >= 30 are miscellaneous and do not write to post models
const (
	MessageSynchronise MessageType = 30 + iota
	MessageReclaim

	// Send new post ID to client
	MessagePostID

	// Concatenation of multiple websocket messages to reduce transport overhead
	MessageConcat

	// Message from the client meant to invoke no operation. Mostly used as a
	// one way ping, because the JS Websocket API does not provide access to
	// pinging.
	MessageNOOP
)

var (
	isTest bool

	errInvalidStructure = errors.New("invalid message structure")
	errInValidCaptcha   = errors.New("invalid captcha provided")

	// Lookup table for message handlers
	handlers = map[MessageType]handler{
		MessageSynchronise:  synchronise,
		MessageReclaim:      reclaimPost,
		MessageInsertThread: insertThread,
		MessageAppend:       appendRune,
		MessageBackspace:    backspace,
		MessageClosePost:    closePost,
		MessageSplice:       spliceText,
		MessageInsertPost:   insertPost,
		MessageInsertImage:  insertImage,
		MessageNOOP:         noop,
	}
)

type handler func([]byte, *Client) error

// Decode message JSON into the supplied type. Will augment, once we switch to
// a binary message protocol.
func decodeMessage(data []byte, dest interface{}) error {
	return json.Unmarshal(data, dest)
}

// EncodeMessage encodes a message for sending through websockets or writing to
// the replication log.
func EncodeMessage(typ MessageType, msg interface{}) ([]byte, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	return prependMessageType(typ, data), nil
}

// Prepend the encoded websocket message type to an already encoded message
func prependMessageType(typ MessageType, data []byte) []byte {
	encoded := make([]byte, len(data)+2)
	typeString := strconv.FormatUint(uint64(typ), 10)

	// Ensure type string is always 2 chars long
	if len(typeString) == 1 {
		encoded[0] = '0'
		encoded[1] = typeString[0]
	} else {
		copy(encoded, typeString)
	}

	copy(encoded[2:], data)

	return encoded
}

// No operation message handler. Used as a one way pseudo-ping.
func noop(_ []byte, _ *Client) error {
	return nil
}
