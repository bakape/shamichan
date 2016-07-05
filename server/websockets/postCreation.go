package websockets

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/aquilax/tripcode"
	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/types"
)

const (
	maxLengthName         = 50
	maxLengthEMail        = 100
	maxLengthAuth         = 50
	maxLengthPostPassword = 50
	maxLengthSubject      = 50
	maxLengthBody         = 2000
)

var (
	errNameTooLong         = errInvalidMessage("name too long")
	errPostPasswordTooLong = errInvalidMessage("password too long")
	errEmailTooLong        = errInvalidMessage("email too long")
	errBodyTooLong         = errInvalidMessage("post body too long")
	errSubjectTooLong      = errInvalidMessage("subject too long")
)

// Insert a new thread into the database
func insertThread(data []byte, c *Client) error {
	var req types.ThreadCreationRequest
	if err := decodeMessage(data, &req); err != nil {
		return err
	}
	if !auth.IsNonMetaBoard(req.Board) {
		return errInvalidBoard
	}

	var tooLong string
	switch {
	case len(req.Name) > maxLengthName:
		tooLong = "name"
	case len(req.Email) > maxLengthEMail:
		tooLong = "email"
	case len(req.Auth) > maxLengthAuth:
		tooLong = "staff title"
	case len(req.Password) > maxLengthPostPassword:
		tooLong = "password"
	case len(req.Subject) > maxLengthSubject:
		tooLong = "subject"
	case len(req.Body) > maxLengthBody:
		tooLong = "post body"
	}
	if tooLong != "" {
		return fmt.Errorf("%s too long", tooLong)
	}

	// TODO: Thread creation cooldown

	// now := time.Now().Unix() * 1000
	// thread := types.DatabaseThread{
	// 	Board: "board",
	// 	BumpTime: now,
	// 	ReplyTime: now,
	// 	Subject: req.Subject,
	// 	Board: req.Board,
	// }
	// post := types.Post{
	// 	Time: now,
	// 	IP: c.ident.IP,
	// 	Board: req.Board,
	// 	Password: req.Password,
	// 	Email: req.Email,
	// }

	return nil
}

// Parse the name field into a name and tripcode, if any
func parseName(name, board string) (string, string, error) {
	// TODO: R/a/dio name swapping

	if name == "" {
		return name, name, nil
	}

	var forcedAnon bool
	q := db.GetBoardConfig(board).Field("forcedAnon").Default(false)
	if err := db.One(q, &forcedAnon); err != nil {
		return "", "", err
	}
	if forcedAnon {
		return "", "", nil
	}

	// #password for tripcodes and ##password for secure tripcodes
	firstHash := strings.IndexByte(name, '#')
	if firstHash > -1 {
		password := name[firstHash+1:]
		name = stripPsuedoWhitespace(name[:firstHash])
		if password[0] == '#' {
			trip := tripcode.SecureTripcode(password[1:], config.Get().Salt)
			return name, trip, nil
		}
		return name, tripcode.Tripcode(password), nil
	}

	return name, "", nil
}

// Strip white-space like unicode characters from srings to avoid "faking"
// spaces
func stripPsuedoWhitespace(s string) string {
	buf := bytes.NewBuffer(make([]byte, 0, len(s)))
	for _, r := range s {
		if r >= 0x2000 && r <= 0x206f {
			if r <= 0x200f || r >= 0x205f || (r >= 0x202a && r <= 0x202f) {
				continue
			}
		}
		buf.WriteRune(r)
	}

	return buf.String()
}
