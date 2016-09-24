// Board creation and configuration and global server administration

package websockets

import (
	"errors"
	"regexp"
	"time"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
)

var (
	errAccessDenied = errors.New("access denied")

	boardNameValidation = regexp.MustCompile(`^[a-z0-9]{1,3}$`)
)

type boardCreationRequest struct {
	Name  string `json:"name"`
	Title string `json:"title"`
	types.Captcha
}

// Board creation request responses
const (
	boardCreated = iota
	invalidBoardName
	boardNameTaken
	titleTooLong
	invalidBoardCreationCaptcha
)

// Answer the admin account's requests for the current server configuration or
// set the server configuration to match the one sent from the admin account.
func configServer(data []byte, c *Client) error {
	if c.UserID != "admin" {
		return errAccessDenied
	}
	if len(data) == 0 { // Request to send current configs
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

	var code int
	switch {
	case !boardNameValidation.MatchString(req.Name):
		code = invalidBoardName
	case len(req.Title) > 100:
		code = titleTooLong
	case !authenticateCaptcha(req.Captcha, c.IP):
		code = invalidBoardCreationCaptcha
	}
	if code > 0 {
		return c.sendMessage(messageCreateBoard, code)
	}

	q := r.Table("boards").Insert(config.DatabaseBoardConfigs{
		Created: time.Now(),
		BoardConfigs: config.BoardConfigs{
			ID:        req.Name,
			Title:     req.Title,
			Spoiler:   "default.jpg",
			Eightball: config.EightballDefaults,
			Banners:   []string{},
			Staff: map[string][]string{
				"owners": []string{c.UserID},
			},
		},
	})
	if err := db.Write(q); r.IsConflictErr(err) {
		return c.sendMessage(messageCreateBoard, boardNameTaken)
	} else if err != nil {
		return err
	}

	// Need to update the config struct separatly
	q = db.GetMain("config").Update(map[string]r.Term{
		"boards": r.Row.Field("boards").Append(req.Name),
	})
	if err := db.Write(q); err != nil {
		return err
	}

	return c.sendMessage(messageCreateBoard, boardCreated)
}
