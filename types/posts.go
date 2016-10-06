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
	BumpTime  int64  `json:"bumpTime" gorethink:"bumpTime"`
	ReplyTime int64  `json:"replyTime" gorethink:"replyTime"`
	Board     string `json:"board" gorethink:"board"`
	Subject   string `json:"subject" gorethink:"subject"`
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
	BumpTime  int64  `json:"bumpTime" gorethink:"bumpTime"`
	ReplyTime int64  `json:"replyTime" gorethink:"replyTime"`
	Subject   string `json:"subject" gorethink:"subject"`
	Posts     []Post `json:"posts" gorethink:"posts"`
}

// DatabaseThread is a template for writing new threads to the database
type DatabaseThread struct {
	PostCtr   int16  `gorethink:"postCtr"`
	ImageCtr  int16  `gorethink:"imageCtr"`
	ID        int64  `gorethink:"id"`
	BumpTime  int64  `gorethink:"bumpTime"`
	ReplyTime int64  `gorethink:"replyTime"`
	Subject   string `gorethink:"subject"`
	Board     string `gorethink:"board"`
}

// Post is a generic post exposed publically through the JSON API. Either OP or
// reply.
type Post struct {
	Editing     bool      `json:"editing" gorethink:"editing"`
	ID          int64     `json:"id" gorethink:"id"`
	OP          int64     `json:"op,omitempty" gorethink:"op"`
	Time        int64     `json:"time" gorethink:"time"`
	LastUpdated int64     `json:"lastUpdated" gorethink:"lastUpdated"`
	Board       string    `json:"board" gorethink:"board"`
	Body        string    `json:"body" gorethink:"body"`
	Name        string    `json:"name,omitempty" gorethink:"name,omitempty"`
	Trip        string    `json:"trip,omitempty" gorethink:"trip,omitempty"`
	Auth        string    `json:"auth,omitempty" gorethink:"auth,omitempty"`
	Email       string    `json:"email,omitempty" gorethink:"email,omitempty"`
	Image       *Image    `json:"image,omitempty" gorethink:"image,omitempty"`
	Backlinks   LinkMap   `json:"backlinks,omitempty" gorethink:"backlinks,omitempty"`
	Links       LinkMap   `json:"links,omitempty" gorethink:"links,omitempty"`
	Commands    []Command `json:"commands,omitempty" gorethink:"commands,omitempty"`
}

// DatabasePost is for writing new posts to a database. It contains the IP and
// Password fields, which are never exposed publically through Post.
type DatabasePost struct {
	Post
	IP       string   `gorethink:"ip"`
	Password []byte   `gorethink:"password"`
	Log      [][]byte `gorethink:"log"`
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
