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

	boardNameValidation = regexp.MustCompile(`^[a-z0-9]{1,3}$`)
)

type boardCreationRequest struct {
	Name, Title string
	types.Captcha
}

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
	case req.Name == "id", !boardNameValidation.MatchString(req.Name):
		code = invalidBoardName
	case len(req.Title) > 100:
		code = titleTooLong
	case !authenticateCaptcha(req.Captcha, c.IP):
		code = invalidBoardCreationCaptcha
	}
	if code > 0 {
		return c.sendMessage(MessageCreateBoard, code)
	}

	q := r.Table("boards").Insert(config.DatabaseBoardConfigs{
		Created: time.Now(),
		BoardConfigs: config.BoardConfigs{
			BoardPublic: config.BoardPublic{
				Title:   req.Title,
				Spoiler: "default.jpg",
				Banners: []string{},
			},
			ID:        req.Name,
			Eightball: config.EightballDefaults,
			Staff: map[string][]string{
				"owners": []string{c.UserID},
			},
		},
	})
	if err := db.Write(q); r.IsConflictErr(err) {
		return c.sendMessage(MessageCreateBoard, boardNameTaken)
	} else if err != nil {
		return err
	}

	return c.sendMessage(MessageCreateBoard, boardCreated)
}
