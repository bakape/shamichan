// Package types contains common shared types used throughout the project
package types

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// CommandType are the various struct types of hash commands and their
// responses, such as dice rolls, #flip, #8ball, etc.
type CommandType uint8

const (
	// Dice is the dice roll command type
	Dice CommandType = iota

	// Flip is the coinflip command type
	Flip

	// EightBall is the the #8ball random answer dispenser command type
	EightBall

	// SyncWatch is the syncronised timer command type for syncronising episode
	// time during group anime watching and such
	SyncWatch

	// Pyu - don't ask
	Pyu

	// Pcount - don't ask
	Pcount
)

// Board stores board metadata and the OPs of all threads
type Board struct {
	Ctr     int64         `json:"ctr"`
	Threads []BoardThread `json:"threads"`
}

// MarshalJSON ensures b.Threads is marshalled to a JSON array even when nil
func (b *Board) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, `{"ctr":%d,"threads":`, b.Ctr)

	if b.Threads == nil {
		buf.WriteString("[]}")
		return buf.Bytes(), nil
	}

	data, err := json.Marshal(b.Threads)
	if err != nil {
		return nil, err
	}
	buf.Write(data)
	buf.WriteRune('}')
	return buf.Bytes(), nil
}

// BoardThread is a stripped down version of Thread for whole-board retrieval
// queries. Reduces server memory usage and served JSON payload.
type BoardThread struct {
	Locked    bool   `json:"locked,omitempty" gorethink:"locked"`
	Archived  bool   `json:"archived,omitempty" gorethink:"archived"`
	Sticky    bool   `json:"sticky,omitempty" gorethink:"sticky"`
	PostCtr   int16  `json:"postCtr" gorethink:"postCtr"`
	ImageCtr  int16  `json:"imageCtr" gorethink:"imageCtr"`
	ID        int64  `json:"id" gorethink:"id"`
	Time      int64  `json:"time" gorethink:"time"`
	Name      string `json:"name,omitempty" gorethink:"name,omitempty"`
	Trip      string `json:"trip,omitempty" gorethink:"trip,omitempty"`
	Auth      string `json:"auth,omitempty" gorethink:"auth,omitempty"`
	Email     string `json:"email,omitempty" gorethink:"email,omitempty"`
	Image     *Image `json:"image,omitempty" gorethink:"image,omitempty"`
	LogCtr    int64  `json:"logCtr" gorethink:"logCtr"`
	BumpTime  int64  `json:"bumpTime" gorethink:"bumpTime"`
	ReplyTime int64  `json:"replyTime" gorethink:"replyTime"`
	Board     string `json:"board" gorethink:"board"`
	Subject   string `json:"subject,omitempty" gorethink:"subject"`
}

// Thread is a transport/export wrapper that stores both the thread metada, its
// opening post data and its contained posts. The composite type itself is not
// stored in the database.
type Thread struct {
	Locked   bool  `json:"locked,omitempty" gorethink:"locked"`
	Archived bool  `json:"archived,omitempty" gorethink:"archived"`
	Sticky   bool  `json:"sticky,omitempty" gorethink:"sticky"`
	PostCtr  int16 `json:"postCtr" gorethink:"postCtr"`
	ImageCtr int16 `json:"imageCtr" gorethink:"imageCtr"`
	Post
	LogCtr    int64          `json:"logCtr" gorethink:"logCtr"`
	BumpTime  int64          `json:"bumpTime" gorethink:"bumpTime"`
	ReplyTime int64          `json:"replyTime" gorethink:"replyTime"`
	Board     string         `json:"board" gorethink:"board"`
	Subject   string         `json:"subject,omitempty" gorethink:"subject"`
	Posts     map[int64]Post `json:"posts,omitempty" gorethink:"posts"`
}

// DatabaseThread is a template for writing new threads to the database
type DatabaseThread struct {
	PostCtr   int16                  `gorethink:"postCtr"`
	ImageCtr  int16                  `gorethink:"imageCtr"`
	ID        int64                  `gorethink:"id"`
	BumpTime  int64                  `gorethink:"bumpTime"`
	ReplyTime int64                  `gorethink:"replyTime"`
	Subject   string                 `gorethink:"subject,omitempty"`
	Board     string                 `gorethink:"board"`
	Posts     map[int64]DatabasePost `gorethink:"posts"`
	Log       [][]byte               `gorethink:"log"`
}

// Post is a generic post exposed publically through the JSON API. Either OP or
// reply.
type Post struct {
	Editing   bool      `json:"editing" gorethink:"editing"`
	ID        int64     `json:"id" gorethink:"id"`
	Time      int64     `json:"time" gorethink:"time"`
	Body      string    `json:"body" gorethink:"body"`
	Name      string    `json:"name,omitempty" gorethink:"name,omitempty"`
	Trip      string    `json:"trip,omitempty" gorethink:"trip,omitempty"`
	Auth      string    `json:"auth,omitempty" gorethink:"auth,omitempty"`
	Email     string    `json:"email,omitempty" gorethink:"email,omitempty"`
	Image     *Image    `json:"image,omitempty" gorethink:"image,omitempty"`
	Backlinks LinkMap   `json:"backlinks,omitempty" gorethink:"backlinks,omitempty"`
	Links     LinkMap   `json:"links,omitempty" gorethink:"links,omitempty"`
	Commands  []Command `json:"commands,omitempty" gorethink:"commands,omitempty"`
}

// DatabasePost is for writing new posts to a database. It contains the IP and
// Password fields, which are never exposed publically through Post.
type DatabasePost struct {
	Post
	IP       string `json:"-" gorethink:"ip"`
	Password []byte `json:"-" gorethink:"password"`
}

// StandalonePost is an extension of Post for serving through `/json/post/:id`.
// Regular posts do not contain or need "op" or "board" fields, because they are
// always retrieved in a known thread context. StandalonePost has these fields
// to allow serving post parenthood for random posts of unknown parenthood.
type StandalonePost struct {
	Post
	OP    int64  `json:"op" gorethink:"op"`
	Board string `json:"board" gorethink:"board"`
}

// LinkMap contains a map of post numbers, this tread is linking, to
// corresponding Link structs
type LinkMap map[int64]Link

// Link stores the target post's parent board and parent thread
type Link struct {
	OP    int64  `json:"op" gorethink:"op"`
	Board string `json:"board" gorethink:"board"`
}

// Command contains the type and value array of hash commands, such as dice
// rolls, #flip, #8ball, etc. The Val field depends on the Type field.
// Dice: []uint16
// Flip: bool
// EightBall: string
// SyncWatch: TODO: SyncWatch storage type
// Pyu: int64
// Pcount: int64
type Command struct {
	Type CommandType `json:"type" gorethink:"type"`
	Val  interface{} `json:"val" gorethink:"val"`
}
