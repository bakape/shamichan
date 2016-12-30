//go:generate ffjson --nodecoder --force-regenerate $GOFILE

// Package common contains common shared types, variables and constants used
// throughout the project
package common

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

// Board is an array stripped down version of Thread for whole-board retrieval
// queries. Reduces server memory usage and served JSON payload.
type Board []BoardThread

// BoardThread is a stripped down version of Thread for board catalog queries
type BoardThread struct {
	ThreadCommon
	ID    uint64 `json:"id"`
	Time  int64  `json:"time"`
	Name  string `json:"name,omitempty"`
	Trip  string `json:"trip,omitempty"`
	Auth  string `json:"auth,omitempty"`
	Image *Image `json:"image,omitempty"`
}

// ThreadCommon contains common fields of both BoardThread and Thread
type ThreadCommon struct {
	Locked    bool   `json:"locked,omitempty"`
	Archived  bool   `json:"archived,omitempty"`
	Sticky    bool   `json:"sticky,omitempty"`
	PostCtr   uint32 `json:"postCtr"`
	ImageCtr  uint32 `json:"imageCtr"`
	ReplyTime int64  `json:"replyTime"`
	LogCtr    uint64 `json:"logCtr"`
	Subject   string `json:"subject"`
	Board     string `json:"board"`
}

// Thread is a transport/export wrapper that stores both the thread metadata,
// its opening post data and its contained posts. The composite type itself is
// not stored in the database.
type Thread struct {
	Abbrev bool `json:"abbrev,omitempty"`
	Post
	ThreadCommon
	Posts []Post `json:"posts"`
}

// DatabaseThread is a template for writing new threads to the database
type DatabaseThread struct {
	ID                uint64
	PostCtr, ImageCtr uint32
	ReplyTime         int64
	Subject, Board    string
	Log               [][]byte
}

// Post is a generic post exposed publically through the JSON API. Either OP or
// reply.
type Post struct {
	Editing   bool      `json:"editing,omitempty"`
	Deleted   bool      `json:"deleted,omitempty"`
	ID        uint64    `json:"id"`
	Time      int64     `json:"time"`
	Body      string    `json:"body"`
	Name      string    `json:"name,omitempty"`
	Trip      string    `json:"trip,omitempty"`
	Auth      string    `json:"auth,omitempty"`
	Backlinks LinkMap   `json:"backlinks,omitempty"`
	Links     LinkMap   `json:"links,omitempty"`
	Commands  []Command `json:"commands,omitempty"`
	Image     *Image    `json:"image,omitempty"`
}

// StandalonePost is a post view that includes the "op" and "board" fields,
// which are not exposed though Post, but are required for retrieving a post
// with unknown parenthood.
type StandalonePost struct {
	Post
	OP    uint64 `json:"op"`
	Board string `json:"board"`
}

// DatabasePost is for writing new posts to a database. It contains the Password
// field, which is never exposed publically through Post.
type DatabasePost struct {
	StandalonePost
	Password []byte
}

// LinkMap contains a map of target post ID: parenthood
type LinkMap map[uint64]Link

// Link stores the target post's parent board and parent thread
type Link struct {
	OP    uint64 `json:"op"`
	Board string `json:"board"`
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
	Type CommandType `json:"type"`
	Val  interface{} `json:"val"`
}
