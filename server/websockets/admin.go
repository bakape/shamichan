// Board creation and configuration and global server administration

package websockets

import (
	"errors"

	"github.com/bakape/meguca/config"
	r "github.com/dancannon/gorethink"

	"github.com/bakape/meguca/db"
)

var (
	errAccessDenied = errors.New("access denied")
)

type boardCreationRequest struct {
	Name  string `json:"name"`
	Title string `json:"title"`
}

// Board creation request responses
const (
	boardCreated = iota
	boardNameTaken
	titleTooLong
)

// Answer the admin account's requests for the current server configuration or
// set the server configuration to match the one sent from the admin account.
func configServer(data []byte, c *Client) error {
	if c.userID != "admin" {
		return errAccessDenied
	}
	if string(data) == "null" { // Request to send current configs
		return c.sendMessage(messageConfigServer, config.Get())
	}

	var conf config.Configs
	if err := decodeMessage(data, &conf); err != nil {
		return err
	}

	query := db.GetMain("config").
		Replace(func(doc r.Term) r.Term {
			return r.Expr(conf).
				// Client can not set boards, so don't update this field
				Merge(map[string]interface{}{
					"id":     "config",
					"boards": doc.Field("boards"),
				})
		})
	if err := db.Write(query); err != nil {
		return err
	}

	return c.sendMessage(messageConfigServer, true)
}

// Handle requests to create a board
func createBoard(data []byte, c *Client) error {
	if !c.isLoggedIn() {
		return errNotLoggedIn
	}

	var req boardCreationRequest
	if err := decodeMessage(data, &req); err != nil {
		return err
	}
	if len(req.Title) > 100 {
		return c.sendMessage(messageCreateBoard, titleTooLong)
	}

	q := r.Table("boards").Insert(config.BoardConfigs{
		ID:        req.Name,
		Title:     req.Title,
		Spoiler:   "default.jpg",
		Eightball: config.EightballDefaults,
		Staff: map[string][]string{
			"owners": []string{c.userID},
		},
	})
	if err := db.Write(q); r.IsConflictErr(err) {
		return c.sendMessage(messageCreateBoard, boardNameTaken)
	} else if err != nil {
		return err
	}

	return c.sendMessage(messageCreateBoard, boardCreated)
}
