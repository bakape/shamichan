package websockets

import (
	"encoding/json"
	"errors"
	"unicode/utf8"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/parser"
)

var (
	errNoPostOpen          = errors.New("no post open")
	errLineEmpty           = errors.New("line empty")
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
	Start int `json:"start"`
	Len   int `json:"len"`
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
func (c *Client) appendRune(data []byte) error {
	if has, err := c.hasPost(); err != nil {
		return err
	} else if !has {
		return nil
	}
	if c.post.len+1 > common.MaxLenBody {
		return common.ErrBodyTooLong
	}
	var char rune
	if err := decodeMessage(data, &char); err != nil {
		return err
	}

	if char == '\n' {
		return c.parseLine(true)
	}

	if err := db.AppendBody(c.post.id, c.post.op, char); err != nil {
		return err
	}
	c.post.WriteRune(char)
	c.post.len++

	return nil
}

// Parse line contents and commit newline. If line contains hash commands or
// links to other posts also commit those and generate backlinks, if needed.
// Appending the newline can be optionally omitted, to optimise post closing
// and similar.
func (c *Client) parseLine(insertNewline bool) error {
	links, comm, err := parser.ParseLine(c.post.LastLine(), c.post.board)
	if err != nil {
		return err
	}
	c.post.WriteRune('\n')
	c.post.len++

	switch {
	case comm.Val != nil:
		err = c.writeCommand(comm)
	case links != nil:
		err = c.writeLinks(links)
	}
	if err != nil {
		return err
	}

	if insertNewline {
		err := db.AppendBody(c.post.id, c.post.op, '\n')
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) writeCommand(comm common.Command) error {
	return db.InsertCommand(c.post.id, c.post.op, comm)
}

func (c *Client) writeLinks(links [][2]uint64) error {
	return db.InsertLinks(c.post.id, c.post.op, links)
}

// Remove one character from the end of the line in the open post
func (c *Client) backspace() error {
	if has, err := c.hasPost(); err != nil {
		return err
	} else if !has {
		return nil
	}
	if len(c.post.LastLine()) == 0 {
		return errLineEmpty
	}
	_, lastRuneLen := utf8.DecodeLastRune(c.post.Bytes())
	c.post.Truncate(c.post.Len() - lastRuneLen)
	c.post.len--

	return db.Backspace(c.post.id, c.post.op)
}

// Close an open post and parse the last line, if needed.
func (c *Client) closePost() error {
	if c.post.id == 0 {
		return errNoPostOpen
	}
	if len(c.post.LastLine()) != 0 {
		if err := c.parseLine(false); err != nil {
			return err
		}
	}
	if err := db.ClosePost(c.post.id, c.post.op); err != nil {
		return err
	}

	c.post = openPost{}
	return nil
}

// Splice the current line's text in the open post. This call is also used for
// text pastes.
func (c *Client) spliceText(data []byte) error {
	if has, err := c.hasPost(); err != nil {
		return err
	} else if !has {
		return nil
	}

	var req spliceRequest
	oldLength := len(string(c.post.LastLine()))
	err := decodeMessage(data, &req)
	switch {
	case err != nil:
		return err
	case req.Start < 0, req.Start+req.Len > oldLength:
		return errInvalidSpliceCoords
	case req.Len == 0 && len(req.Text) == 0:
		return errSpliceNOOP // This does nothing. Client-side error.
	case len(req.Text) > common.MaxLenBody:
		return errSpliceTooLong // Nice try, kid
	}

	return c.spliceLine(req)
}

// Splice the first line of the text. If there are more lines, parse the
// previous one and recurse until all lines are parsed.
func (c *Client) spliceLine(req spliceRequest) error {
	var (
		old   = []rune(string(c.post.LastLine()))
		start = old[:req.Start]
		end   []rune
	)

	// -1 has special meaning - slice off till line end.
	if req.Len < 0 {
		end = req.Text
		c.post.len += len(req.Text) - len(old[req.Start:])
	} else {
		end = append(req.Text, old[req.Start+req.Len:]...)
		c.post.len += -req.Len + len(req.Text)
	}
	res := spliceMessage{
		ID: c.post.id,
		spliceRequestString: spliceRequestString{
			spliceCoords: req.spliceCoords,
			Text:         string(req.Text),
		},
	}

	// Slice until newline, if any, and delay the next line's splicing until the
	// next recursive spliceLine call
	var (
		delayed      []rune
		firstNewline = -1
	)
	for i, r := range end { // Find first newline
		if r == '\n' {
			firstNewline = i
			break
		}
	}
	if firstNewline != -1 {
		delayed = end[firstNewline+1:]
		end = end[:firstNewline]
		res.Len = -1 // Special meaning. Client should replace till line end.
		res.Text = string(end)
		c.post.len -= len(delayed) + 1
	}

	// Goes over max post length. Trim the end.
	exceeding := c.post.len - common.MaxLenBody
	if exceeding > 0 {
		end = end[:len(end)-exceeding]
		res.Len = -1
		res.Text = string(end)
		c.post.len = common.MaxLenBody
	}

	c.post.TrimLastLine()
	c.post.WriteString(string(append(start, end...)))

	msg, err := common.EncodeMessage(common.MessageSplice, res)
	if err != nil {
		return err
	}
	err = db.SplicePost(c.post.id, c.post.op, msg, c.post.String())
	if err != nil {
		return err
	}

	// Recurse on the text sliced after the newline, until all lines are
	// processed. Unless the text body is already over max length
	if delayed != nil && exceeding < 0 {
		if err := c.parseLine(true); err != nil {
			return err
		}
		return c.spliceLine(spliceRequest{Text: delayed})
	}
	return nil
}

// Insert and image into an existing open post
func (c *Client) insertImage(data []byte) error {
	if has, err := c.hasPost(); err != nil {
		return err
	} else if !has {
		return nil
	}
	if c.post.hasImage {
		return errHasImage
	}

	var req ImageRequest
	if err := decodeMessage(data, &req); err != nil {
		return err
	}

	if config.GetBoardConfigs(c.post.board).TextOnly {
		return errTextOnly
	}

	img, err := getImage(req.Token, req.Name, req.Spoiler)
	if err != nil {
		return err
	}
	c.post.hasImage = true

	return db.InsertImage(c.post.id, c.post.op, *img)
}
