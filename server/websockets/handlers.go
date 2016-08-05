// Wecbsocket message handlers central file

package websockets

import (
	"bufio"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/types"
)

// identifier codes for websocket message types
type messageType uint8

// 1 - 29 modify post model state
const (
	messageInvalid messageType = iota
	messageInsertThread
	messageInsertPost
	messageAppend
	messageBackspace
	messageSplice
	messageInsertLine
	messageClosePost
	messageBacklink
	messageCommand
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
	messageChangePassword

	// Board administration
	messageConfigServer
	messageCreateBoard
	messageConfigBoard
)

var (
	isTest bool

	errInvalidStructure = errors.New("invalid message structure")
	errInValidCaptcha   = errors.New("no captcha provided")

	// Lookup table for message handlers
	handlers = map[messageType]handler{
		messageSynchronise:    synchronise,
		messageResynchronise:  resynchronise,
		messageRegister:       register,
		messageLogin:          login,
		messageAuthenticate:   authenticateSession,
		messageLogout:         logOut,
		messageLogOutAll:      logOutAll,
		messageChangePassword: changePassword,
		messageConfigServer:   configServer,
		messageCreateBoard:    createBoard,
		messageConfigBoard:    configBoard,
		messageInsertThread:   insertThread,
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

// Decode message JSON into the suplied type
func decodeMessage(data []byte, dest interface{}) error {
	err := json.Unmarshal(data, dest)
	if err != nil {
		return errInvalidStructure
	}
	return nil
}

// Post a request to the SolveMedia API to authenticate a captcha
func authenticateCaptcha(captcha types.Captcha, ip string) bool {
	conf := config.Get()

	// Captchas disablled or running tests. Can not use API, when testing
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
		printCapthcaError(err)
		return false
	}
	defer res.Body.Close()

	reader := bufio.NewReader(res.Body)
	status, err := reader.ReadString('\n')
	if err != nil {
		printCapthcaError(err)
		return false
	}
	if status[:len(status)-1] != "true" {
		reason, _ := reader.ReadString('\n')
		printCapthcaError(errors.New(reason[:len(reason)-1]))
		return false
	}

	return true
}

func printCapthcaError(err error) {
	log.Println(errCaptcha{err})
}
