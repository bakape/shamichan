// Board creation and configuration and global server administration

package websockets

import (
	"errors"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	r "github.com/dancannon/gorethink"
)

// Board creation request responses
const (
	boardCreated = iota
	invalidBoardName
	boardNameTaken
	titleTooLong
	invalidBoardCreationCaptcha
)

var (
	errAccessDenied = errors.New("access denied")
	errNotLoggedIn  = errors.New("not logged in")
)

// Answer the admin account's requests for the current server configuration or
// set the server configuration to match the one sent from the admin account.
func configServer(data []byte, c *Client) error {
	if c.UserID != "admin" {
		return errAccessDenied
	}
	if len(data) == 0 { // Request to send current configs
		return c.sendMessage(MessageConfigServer, config.Get())
	}

	var conf config.Configs
	if err := decodeMessage(data, &conf); err != nil {
		return err
	}

	query := db.GetMain("config").
		Replace(func(doc r.Term) r.Term {
			return r.Expr(conf).Merge(map[string]string{
				"id": "config",
			})
		})
	if err := db.Write(query); err != nil {
		return err
	}

	return c.sendMessage(MessageConfigServer, true)
}
