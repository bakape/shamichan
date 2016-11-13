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

	// Flip is the coin flip command type
	Flip

	// EightBall is the the #8ball random answer dispenser command type
	EightBall

	// SyncWatch is the synchronized timer command type for synchronizing
	// episode time during group anime watching and such
	SyncWatch

	// Pyu - don't ask
	Pyu

	// Pcount - don't ask
	Pcount
)

// Board stores board metadata and the OPs of all threads
type Board struct {
	Ctr     int64        `json:"ctr"`
	Threads BoardThreads `json:"threads"`
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

// BoardThreads is an array stripped down version of Thread for whole-board
// retrieval queries. Reduces server memory usage and served JSON payload.
type BoardThreads []struct {
	Locked      bool   `json:"locked,omitempty" gorethink:"locked"`
	Archived    bool   `json:"archived,omitempty" gorethink:"archived"`
	Sticky      bool   `json:"sticky,omitempty" gorethink:"sticky"`
	PostCtr     int16  `json:"postCtr" gorethink:"postCtr"`
	ImageCtr    int16  `json:"imageCtr" gorethink:"imageCtr"`
	ID          int64  `json:"id" gorethink:"id"`
	Time        int64  `json:"time" gorethink:"time"`
	LastUpdated int64  `json:"lastUpdated" gorethink:"lastUpdated"`
	Name        string `json:"name,omitempty" gorethink:"name,omitempty"`
	Trip        string `json:"trip,omitempty" gorethink:"trip,omitempty"`
	Auth        string `json:"auth,omitempty" gorethink:"auth,omitempty"`
	Email       string `json:"email,omitempty" gorethink:"email,omitempty"`
	Image       *Image `json:"image,omitempty" gorethink:"image,omitempty"`
	ReplyTime   int64  `json:"replyTime" gorethink:"replyTime"`
	Board       string `json:"board" gorethink:"board"`
	Subject     string `json:"subject" gorethink:"subject"`
}

// Thread is a transport/export wrapper that stores both the thread metadata,
// its opening post data and its contained posts. The composite type itself is
// not stored in the database.
type Thread struct {
	Locked   bool  `json:"locked,omitempty" gorethink:"locked"`
	Archived bool  `json:"archived,omitempty" gorethink:"archived"`
	Sticky   bool  `json:"sticky,omitempty" gorethink:"sticky"`
	PostCtr  int16 `json:"postCtr" gorethink:"postCtr"`
	ImageCtr int16 `json:"imageCtr" gorethink:"imageCtr"`
	Post
	ReplyTime   int64  `json:"replyTime" gorethink:"replyTime"`
	LastUpdated int64  `json:"lastUpdated" gorethink:"lastUpdated"`
	Subject     string `json:"subject" gorethink:"subject"`
	Board       string `json:"board" gorethink:"board"`
	Posts       []Post `json:"posts" gorethink:"posts"`
}

// DatabaseThread is a template for writing new threads to the database
type DatabaseThread struct {
	PostCtr   int    `gorethink:"postCtr"`
	ImageCtr  int    `gorethink:"imageCtr"`
	ID        int64  `gorethink:"id"`
	ReplyTime int64  `gorethink:"replyTime"`
	Subject   string `gorethink:"subject"`
	Board     string `gorethink:"board"`
}

// Post is a generic post exposed publically through the JSON API. Either OP or
// reply.
type Post struct {
	Editing   bool      `json:"editing,omitempty" gorethink:"editing"`
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

// StandalonePost is a post view that includes the "op" and "board" fields,
// which are not exposed though Post, but are required for retrieving a post
// with unknown parenthood.
type StandalonePost struct {
	Post
	OP    int64  `json:"op" gorethink:"op"`
	Board string `json:"board" gorethink:"board"`
}

// DatabasePost is for writing new posts to a database. It contains the IP and
// Password fields, which are never exposed publically through Post.
type DatabasePost struct {
	StandalonePost
	IP          string   `gorethink:"ip"`
	Password    []byte   `gorethink:"password"`
	Log         [][]byte `gorethink:"log"`
	LastUpdated int64    `json:"lastUpdated" gorethink:"lastUpdated"`
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
