// Wecbsocket message handlers central file

package websockets

import (
	"bufio"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/types"
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
	MessageDelete
)

// >= 30 are miscellaneous and do not write to post models
const (
	// Update feeds
	MessageSynchronise MessageType = 30 + iota
	MessageReclaim
	MessageSwitchSync

	// Account management
	MessageRegister
	MessageLogin
	MessageAuthenticate
	MessageLogout
	MessageLogOutAll
	MessageChangePassword

	// Board administration
	MessageConfigServer
	MessageCreateBoard

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
	errInValidCaptcha   = errors.New("no captcha provided")

	// Lookup table for message handlers
	handlers = map[MessageType]handler{
		MessageSynchronise:    synchronise,
		MessageReclaim:        reclaimPost,
		MessageRegister:       register,
		MessageLogin:          login,
		MessageAuthenticate:   authenticateSession,
		MessageLogout:         logOut,
		MessageLogOutAll:      logOutAll,
		MessageChangePassword: changePassword,
		MessageConfigServer:   configServer,
		MessageCreateBoard:    createBoard,
		MessageInsertThread:   insertThread,
		MessageAppend:         appendRune,
		MessageBackspace:      backspace,
		MessageClosePost:      closePost,
		MessageSplice:         spliceText,
		MessageInsertPost:     insertPost,
		MessageInsertImage:    insertImage,
		MessageNOOP:           noop,
	}
)

// Error during authenticating a captcha. These are not reported to the client,
// only logged.
type errCaptcha struct {
	error
}

func (e errCaptcha) Error() string {
	return "captcha error: " + e.error.Error()
}

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

// Post a request to the SolveMedia API to authenticate a captcha
func authenticateCaptcha(captcha types.Captcha, ip string) bool {
	conf := config.Get()

	// Captchas disabled or running tests. Can not use API, when testing
	if isTest || !conf.Captcha {
		return true
	}

	if captcha.Captcha == "" || captcha.CaptchaID == "" {
		return false
	}

	data := url.Values{
		"privatekey": {conf.CaptchaPrivateKey},
		"challenge":  {captcha.CaptchaID},
		"response":   {captcha.Captcha},
		"remoteip":   {ip},
	}
	res, err := http.PostForm("http://verify.solvemedia.com/papi/verify", data)
	if err != nil {
		printCaptchaError(err)
		return false
	}
	defer res.Body.Close()

	reader := bufio.NewReader(res.Body)
	status, err := reader.ReadString('\n')
	if err != nil {
		printCaptchaError(err)
		return false
	}
	if status[:len(status)-1] != "true" {
		reason, _ := reader.ReadString('\n')
		printCaptchaError(errors.New(reason[:len(reason)-1]))
		return false
	}

	return true
}

func printCaptchaError(err error) {
	log.Println(errCaptcha{err})
}

// No operation message handler. Used as a one way pseudo-ping.
func noop(_ []byte, _ *Client) error {
	return nil
}
