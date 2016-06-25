// Wecbsocket message handlers central file

package websockets

import "encoding/json"

// identifier codes for websocket message types
type messageType uint8

// 1 - 29 modify post model state
const (
	messageInvalid messageType = iota
	messageInsertThread
	messageInsertPost
)

// >= 30 are miscelenious and do not write to post models
const (
	// Update feeds
	messageSynchronise messageType = 30 + iota
	messageResynchronise
	messageSwitchSync

	// Account management
	messageRegister
	messageLogin
	messageAuthenticate
	messageLogout
	messageLogOutAll
)

type handler func([]byte, *Client) error

// Lookup table for message handlers
var handlers = map[messageType]handler{
	messageSynchronise:   synchronise,
	messageResynchronise: resynchronise,
	messageRegister:      register,
	messageLogin:         login,
	messageAuthenticate:  authenticateSession,
}

// Error while parsing the message. Denotes that either the message does not
// follow the structural spec or contains optional fields in unsupported
// combinations.
type errInvalidMessage string

func (e errInvalidMessage) Error() string {
	return string(e)
}

var (
	errInvalidStructure = errInvalidMessage("Invalid message structure")
)

// Decode message JSON into the suplied type
func decodeMessage(data []byte, dest interface{}) error {
	err := json.Unmarshal(data, dest)
	if err != nil {
		return errInvalidStructure
	}
	return nil
}
