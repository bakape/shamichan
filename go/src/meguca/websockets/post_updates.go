package websockets

import (
	"encoding/json"
	"errors"
	"meguca/auth"
	"meguca/common"
	"meguca/config"
	"meguca/db"
	"meguca/parser"
	"meguca/util"
	"time"
	"unicode/utf8"
)

var (
	errNoPostOpen          = errors.New("no post open")
	errEmptyPost           = errors.New("post body empty")
	errTooManyLines        = errors.New("too many lines in post body")
	errInvalidSpliceCoords = errors.New("invalid splice coordinates")
	errSpliceTooLong       = errors.New("splice text too long")
	errSpliceNOOP          = errors.New("splice NOOP")
	errTextOnly            = errors.New("text only board")
	errHasImage            = errors.New("post already has image")
)

// Like spliceRequest, but with a string Text field. Used for internal
// conversions between []rune and string.
type spliceRequestString struct {
	spliceCoords
	Text string `json:"text"`
}

// Common part of a splice request and a splice response
type spliceCoords struct {
	Start uint `json:"start"`
	Len   uint `json:"len"`
}

// Response to a spliceRequest. Sent to all listening clients.
type spliceMessage struct {
	ID uint64 `json:"id"`
	spliceRequestString
}

// Request or to replace the current line's text starting at an exact position
// in the current line
type spliceRequest struct {
	spliceCoords
	Text []rune
}

// Custom unmarshaling of string -> []rune
func (s *spliceRequest) UnmarshalJSON(buf []byte) error {
	var tmp spliceRequestString
	if err := json.Unmarshal(buf, &tmp); err != nil {
		return err
	}
	*s = spliceRequest{
		spliceCoords: tmp.spliceCoords,
		Text:         []rune(tmp.Text),
	}
	return nil
}

// Custom marshaling of []rune -> string
func (s spliceRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal(spliceRequestString{
		spliceCoords: s.spliceCoords,
		Text:         string(s.Text),
	})
}

// Append a rune to the body of the open post
func (c *Client) appendRune(data []byte) (err error) {
	has, err := c.hasPost()
	switch {
	case err != nil:
		return
	case !has:
		return
	case c.post.len+1 > common.MaxLenBody:
		return common.ErrBodyTooLong
	}

	var char rune
	err = decodeMessage(data, &char)
	switch {
	case err != nil:
		return
	case char == 0:
		return common.ErrContainsNull
	case char == '\n':
		c.post.lines++
		if c.post.lines > common.MaxLinesBody {
			return errTooManyLines
		}
	}

	msg, err := common.EncodeMessage(
		common.MessageAppend,
		[2]uint64{c.post.id, uint64(char)},
	)
	if err != nil {
		return
	}

	c.post.body = append(c.post.body, string(char)...)
	c.post.len++
	return c.updateBody(msg, 1)
}

// Send message to thread update feed and writes the open post's buffer to the
// embedded database. Requires locking of c.openPost.
// n specifies the number of characters updated.
func (c *Client) updateBody(msg []byte, n int) error {
	c.feed.SetOpenBody(c.post.id, c.post.body, msg)
	err := c.incrementSpamScore(time.Duration(n) * auth.CharScore)
	if err != nil {
		return err
	}
	return db.SetOpenBody(c.post.id, c.post.body)
}

// Increment the spam score for this IP by score. If the client requires a new
// solved captcha, send a notification.
func (c *Client) incrementSpamScore(score time.Duration) error {
	exceeds, err := auth.IncrementSpamScore(c.ip, score)
	if err != nil {
		return err
	}
	if exceeds {
		return c.sendMessage(common.MessageCaptcha, 0)
	}
	return nil
}

// Remove one character from the end of the line in the open post
func (c *Client) backspace() error {
	has, err := c.hasPost()
	switch {
	case err != nil:
		return err
	case !has:
		return nil
	case c.post.len == 0:
		return errEmptyPost
	}

	msg, err := common.EncodeMessage(common.MessageBackspace, c.post.id)
	if err != nil {
		return err
	}

	r, lastRuneLen := utf8.DecodeLastRune(c.post.body)
	c.post.body = c.post.body[:len(c.post.body)-lastRuneLen]
	if r == '\n' {
		c.post.lines--
	}
	c.post.len--

	return c.updateBody(msg, 1)
}

// Close an open post and parse the last line, if needed.
func (c *Client) closePost() error {
	if c.post.id == 0 {
		return errNoPostOpen
	}

	var (
		links [][2]uint64
		com   []common.Command
		err   error
	)
	if c.post.len != 0 {
		links, com, err = parser.ParseBody(c.post.body, c.post.board)
		if err != nil {
			return err
		}
	}

	err = db.ClosePost(c.post.id, c.post.op, string(c.post.body), links, com)
	if err != nil {
		return err
	}
	c.post = openPost{}
	return nil
}

// Splice the text in the open post. This call is also used for text pastes.
func (c *Client) spliceText(data []byte) error {
	if has, err := c.hasPost(); err != nil {
		return err
	} else if !has {
		return nil
	}

	// Decode and validate
	var (
		req spliceRequest
		err = decodeMessage(data, &req)
	)
	switch {
	case err != nil:
		return err
	case req.Start > common.MaxLenBody,
		req.Len > common.MaxLenBody,
		int(req.Start+req.Len) > c.post.len:
		return errInvalidSpliceCoords
	case req.Len == 0 && len(req.Text) == 0:
		return errSpliceNOOP // This does nothing. Client-side error.
	case len(req.Text) > common.MaxLenBody:
		return errSpliceTooLong // Nice try, kid
	}

	for _, r := range req.Text {
		if r == 0 {
			return common.ErrContainsNull
		}
	}

	var (
		old = []rune(string(c.post.body))
		end = append(req.Text, old[req.Start+req.Len:]...)
	)
	c.post.len += -int(req.Len) + len(req.Text)
	res := spliceMessage{
		ID: c.post.id,
		spliceRequestString: spliceRequestString{
			spliceCoords: req.spliceCoords,
			Text:         string(req.Text),
		},
	}

	// If it goes over the max post length, trim the end
	exceeding := c.post.len - common.MaxLenBody
	if exceeding > 0 {
		end = end[:len(end)-exceeding]
		res.Len = uint(len(old[int(req.Start):]))
		res.Text = string(end)
		c.post.len = common.MaxLenBody
	}

	msg, err := common.EncodeMessage(common.MessageSplice, res)
	if err != nil {
		return err
	}

	// Need to prevent modifications to the original slice, as there might be
	// concurrent reads in the update feed.
	c.post.body = util.CloneBytes(c.post.body)

	byteStartPos := 0
	for _, r := range old[:req.Start] {
		byteStartPos += utf8.RuneLen(r)
	}
	c.post.body = append(c.post.body[:byteStartPos], string(end)...)

	c.post.countLines()
	if c.post.lines > common.MaxLinesBody {
		return errTooManyLines
	}

	// +1, so you can't spam zero insert splices to infinity
	return c.updateBody(msg, len(res.Text)+1)
}

// Insert and image into an existing open post
func (c *Client) insertImage(data []byte) (err error) {
	has, err := c.hasPost()
	switch {
	case err != nil:
		return
	case !has:
		return errNoPostOpen
	case c.post.hasImage:
		return errHasImage
	}

	var req ImageRequest
	err = decodeMessage(data, &req)
	if err != nil {
		return
	}

	if config.GetBoardConfigs(c.post.board).TextOnly {
		return errTextOnly
	}

	tx, err := db.StartTransaction()
	if err != nil {
		return
	}
	defer db.RollbackOnError(tx, &err)

	img, err := getImage(tx, req.Token, req.Name, req.Spoiler)
	if err != nil {
		return
	}
	c.post.hasImage = true
	c.post.isSpoilered = req.Spoiler

	err = db.InsertImage(tx, c.post.id, *img)
	if err != nil {
		return
	}
	err = tx.Commit()
	if err != nil {
		return
	}

	msg, err := common.EncodeMessage(common.MessageInsertImage, struct {
		ID uint64 `json:"id"`
		common.Image
	}{
		ID:    c.post.id,
		Image: *img,
	})
	if err != nil {
		return
	}
	c.feed.InsertImage(c.post.id, *img, msg)

	return c.incrementSpamScore(auth.ImageScore)
}

// Spoiler an already inserted image in an unclosed post
func (c *Client) spoilerImage() (err error) {
	has, err := c.hasPost()
	switch {
	case err != nil:
		return err
	case !has:
		return errNoPostOpen
	case !c.post.hasImage:
		return errors.New("post does not have an image")
	case c.post.isSpoilered:
		return errors.New("already spoilered")
	}

	err = db.SpoilerImage(c.post.id)
	if err != nil {
		return
	}
	msg, err := common.EncodeMessage(common.MessageSpoiler, c.post.id)
	if err != nil {
		return
	}
	c.feed.SpoilerImage(c.post.id, msg)

	return
}
