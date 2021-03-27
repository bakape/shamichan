package websockets

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/geoip"
	"github.com/bakape/meguca/parser"
	"github.com/bakape/meguca/util"
	"github.com/bakape/meguca/websockets/feeds"
)

var (
	errReadOnly          = common.ErrInvalidInput("read only board")
	errInvalidImageToken = common.ErrInvalidInput("image token")
	errNoTextOrImage     = common.ErrInvalidInput("no text or image")
)

// ThreadCreationRequest contains data for creating a new thread
type ThreadCreationRequest struct {
	ReplyCreationRequest
	Subject, Board string
}

// ReplyCreationRequest contains common fields for both thread and reply
// creation
type ReplyCreationRequest struct {
	Sage, Open bool
	Image      ImageRequest
	auth.SessionCreds
	Name, Password, Body string
}

// ImageRequest contains data for allocating an image
type ImageRequest struct {
	Spoiler     bool
	Token, Name string
}

// CreateThread creates a new tread and writes it to the database.
// open specifies, if the thread OP should stay open after creation.
func CreateThread(req ThreadCreationRequest, ip string) (
	post db.Post, err error,
) {
	if !auth.IsNonMetaBoard(req.Board) {
		err = common.ErrInvalidBoard(req.Board)
		return
	}
	err = db.IsBanned(req.Board, ip)
	if err != nil {
		return
	}
	conf, err := getBoardConfig(req.Board)
	if err != nil {
		return
	}
	post, err = constructPost(req.ReplyCreationRequest, conf, ip)
	if err != nil {
		return
	}
	subject, err := parser.ParseSubject(req.Subject)
	if err != nil {
		return
	}

	// Must ensure image token usage is done atomically, as not to cause
	// possible data races with unused image cleanup
	err = db.InTransaction(false, func(tx *sql.Tx) (err error) {
		err = db.InsertThread(tx, subject, &post)
		if err != nil {
			return
		}

		if !conf.TextOnly && req.Image.Token != "" &&
			req.Image.Name != "" {
			err = insertImage(tx, req.Image, &post)
			if err != nil {
				return
			}
		}
		return
	})

	return
}

// Insert image into a post on post creation
func insertImage(tx *sql.Tx, req ImageRequest, p *db.Post) (err error) {
	formatImageName(&req.Name)

	// TODO: Get rid of this redundant decoding once we switch to a JSON-only
	// application server
	buf, err := db.InsertImage(tx, p.ID, req.Token, req.Name, req.Spoiler)
	if err != nil {
		return
	}
	err = json.Unmarshal(buf, &p.Image)
	if err != nil {
		return
	}

	p.Image.Name = req.Name
	p.Image.Spoiler = req.Spoiler
	return
}

// CreatePost creates a new post and writes it to the database.
// open specifies, if the post should stay open after creation.
func CreatePost(
	op uint64,
	board, ip string,
	req ReplyCreationRequest,
) (
	post db.Post, msg []byte, err error,
) {
	err = db.IsBanned(board, ip)
	if err != nil {
		return
	}

	conf, err := getBoardConfig(board)
	if err != nil {
		return
	}

	// Post must have either at least one character or an image to be allocated
	hasImage := !conf.TextOnly && req.Image.Token != "" && req.Image.Name != ""
	if req.Body == "" && !hasImage {
		err = errNoTextOrImage
		return
	}

	// Assert thread is not locked
	locked, err := db.CheckThreadLocked(op)
	switch {
	case err != nil:
		return
	case locked:
		err = common.StatusError{errors.New("thread is locked"), 400}
		return
	}

	post, err = constructPost(req, conf, ip)
	if err != nil {
		return
	}

	post.OP = op

	// Must ensure image token usage is done atomically, as not to cause
	// possible data races with unused image cleanup
	err = db.InTransaction(false, func(tx *sql.Tx) (err error) {
		err = db.InsertPost(tx, &post)
		if err != nil {
			return
		}

		if hasImage {
			err = insertImage(tx, req.Image, &post)
			if err != nil {
				return
			}
		}

		return
	})

	msg, err = common.EncodeMessage(common.MessageInsertPost, post.Post)
	return
}

// Insert a new post into the database
func (c *Client) insertPost(data []byte) (err error) {
	err = c.closePreviousPost()
	if err != nil {
		return
	}

	needCaptcha, err := db.NeedCaptcha(c.captchaSession, c.ip)
	if err != nil {
		return
	}
	if needCaptcha {
		return c.sendMessage(common.MessageCaptcha, 0)
	}

	var req ReplyCreationRequest
	err = decodeMessage(data, &req)
	if err != nil {
		return
	}
	// Replies created through websockets can only be open
	req.Open = true

	_, op, board := feeds.GetSync(c)
	post, msg, err := CreatePost(op, board, c.ip, req)
	if err != nil {
		return
	}

	// Ensure the client knows the post ID, before the public post insertion
	// update message is sent
	err = c.sendMessage(common.MessagePostID, post.ID)
	if err != nil {
		return
	}

	if post.Editing {
		err = db.SetOpenBody(post.ID, []byte(post.Body))
		if err != nil {
			return
		}
		c.post.init(post.StandalonePost)
	}
	c.feed.InsertPost(post.StandalonePost.Post, msg)
	conf := config.Get()
	c.incrementSpamScore(conf.PostCreationScore +
		conf.CharScore*uint(c.post.len))
	c.setLastTime()
	return
}

// If the client has a previous post, close it silently
func (c *Client) closePreviousPost() error {
	if c.post.id != 0 {
		return c.closePost()
	}
	return nil
}

// Retrieve post-related board configurations
func getBoardConfig(board string) (conf config.BoardConfigs, err error) {
	conf = config.GetBoardConfigs(board).BoardConfigs
	if conf.ReadOnly {
		err = errReadOnly
	}
	return
}

// Construct the common parts of the new post for both threads and replies
func constructPost(
	req ReplyCreationRequest,
	conf config.BoardConfigs,
	ip string,
) (
	post db.Post, err error,
) {
	post = db.Post{
		StandalonePost: common.StandalonePost{
			Post: common.Post{
				Sage: req.Sage,
				Body: req.Body,
			},
			Board: conf.ID,
		},
		IP: ip,
	}

	if !conf.ForcedAnon {
		post.Name, post.Trip, err = parser.ParseName(req.Name)
		if err != nil {
			return
		}
	}

	if conf.Flags {
		post.Flag = geoip.LookUp(ip)
	}

	if utf8.RuneCountInString(req.Body) > common.MaxLenBody {
		err = common.ErrBodyTooLong
		return
	}

	lines := 0
	for _, r := range req.Body {
		if r == '\n' {
			lines++
		}
	}
	if lines > common.MaxLinesBody {
		err = errTooManyLines
		return
	}

	// Attach staff position title after validations
	if req.UserID != "" {
		post.Auth, err = db.FindPosition(conf.ID, req.UserID)
		if err != nil {
			return
		}
		if post.Auth != 0 {
			var loggedIn bool
			loggedIn, err = db.IsLoggedIn(req.UserID, req.Session)
			if err != nil {
				return
			}
			if !loggedIn {
				err = common.ErrInvalidCreds
				return
			}
		}
	}

	if req.Open {
		post.Editing = true

		// Posts that are committed in one action need not a password, as they
		// are closed on commit and can not be reclaimed
		err = parser.VerifyPostPassword(req.Password)
		if err != nil {
			return
		}
		post.Password, err = auth.BcryptHash(req.Password, 4)
		if err != nil {
			return
		}
	} else {
		// TODO: Move DB checks out of the parser. The parser should just parse.
		// Return slices of pointers to links and commands that need to be
		// validated.
		post.Links, post.Commands, err = parser.ParseBody(
			[]byte(req.Body),
			conf.ID,
			post.OP,
			post.ID,
			ip,
			false,
		)
		if err != nil {
			return
		}
	}

	return
}

// Trim on the last dot in the file name, but also strip for .tar.gz and
// .tar.xz as special cases.
func formatImageName(name *string) {
	util.TrimString(name, 200)

	if i := strings.LastIndexByte(*name, '.'); i != -1 {
		*name = (*name)[:i]
		if strings.HasSuffix(*name, ".tar") {
			*name = (*name)[:len(*name)-4]
		}
	}
}
