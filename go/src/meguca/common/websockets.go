package common

import (
	"bytes"
	"encoding/json"
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
	MessageBacklink
	MessageInsertImage
	MessageSpoiler
	MessageDeletePost
	MessageBanned
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

	// Transmit current synced IP count to client
	MessageSyncCount

	// Send current server Unix time to client
	MessageServerTime

	// Redirect the client to a specific board
	MessageRedirect

	// Send a notification to a client
	MessageNotification
)

// Clients exposes the global client map to all packages without causing
// circular imports
var Clients interface {
	Clear()
	GetByIP(ip string) []Client
}

// Client exposes some globally accessible websocket client functionality
// without causing circular imports
type Client interface {
	Redirect(board string)
}

// Feeds exposes update feed storage map without causing circular imports
var Feeds interface {
	SendTo(id uint64, msg []byte)
	ClosePost(id, op uint64, msg []byte)
}

// EncodeMessage encodes a message for sending through websockets or writing to
// the replication log.
func EncodeMessage(typ MessageType, msg interface{}) ([]byte, error) {
	data, err := json.Marshal(msg)
	switch {
	case err != nil:
		return nil, err
	case typ != MessageConcat && bytes.IndexRune(data, 0) != -1:
		return nil, ErrContainsNull
	default:
		return PrependMessageType(typ, data), nil
	}
}

// PrependMessageType prepends the encoded websocket message type to an already
// encoded message
func PrependMessageType(typ MessageType, data []byte) []byte {
	encoded := make([]byte, len(data)+2)

	// Ensure type string is always 2 chars long
	var i int
	if typ < 10 {
		encoded[0] = '0'
		i = 1
	}
	strconv.AppendUint(encoded[i:i], uint64(typ), 10)

	copy(encoded[2:], data)

	return encoded
}
