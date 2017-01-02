//go:generate ffjson --nodecoder $GOFILE

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
	ID    uint64 `json:"id" gorethink:"id"`
	Time  int64  `json:"time" gorethink:"time"`
	Name  string `json:"name,omitempty" gorethink:"name,omitempty"`
	Trip  string `json:"trip,omitempty" gorethink:"trip,omitempty"`
	Auth  string `json:"auth,omitempty" gorethink:"auth,omitempty"`
	Image *Image `json:"image,omitempty" gorethink:"image,omitempty"`
}

// ThreadCommon contains common fields of both BoardThread and Thread
type ThreadCommon struct {
	Locked      bool   `json:"locked,omitempty" gorethink:"locked"`
	Archived    bool   `json:"archived,omitempty" gorethink:"archived"`
	Sticky      bool   `json:"sticky,omitempty" gorethink:"sticky"`
	PostCtr     uint32 `json:"postCtr" gorethink:"postCtr"`
	ImageCtr    uint32 `json:"imageCtr" gorethink:"imageCtr"`
	ReplyTime   int64  `json:"replyTime" gorethink:"replyTime"`
	LastUpdated int64  `json:"lastUpdated" gorethink:"lastUpdated"`
	Subject     string `json:"subject" gorethink:"subject"`
	Board       string `json:"board" gorethink:"board"`
}

// Thread is a transport/export wrapper that stores both the thread metadata,
// its opening post data and its contained posts. The composite type itself is
// not stored in the database.
type Thread struct {
	Abbrev bool `json:"abbrev,omitempty"`
	Post
	ThreadCommon
	Posts []Post `json:"posts" gorethink:"posts"`
}

// DatabaseThread is a template for writing new threads to the database
type DatabaseThread struct {
	PostCtr   uint32 `gorethink:"postCtr"`
	ImageCtr  uint32 `gorethink:"imageCtr"`
	ID        uint64 `gorethink:"id"`
	ReplyTime int64  `gorethink:"replyTime"`
	Subject   string `gorethink:"subject"`
	Board     string `gorethink:"board"`
}

// Post is a generic post exposed publically through the JSON API. Either OP or
// reply.
type Post struct {
	Editing   bool      `json:"editing,omitempty" gorethink:"editing"`
	Deleted   bool      `json:"deleted,omitempty" gorethink:"deleted,omitempty"`
	Banned    bool      `json:"banned,omitempty" gorethink:"banned,omitempty"`
	ID        uint64    `json:"id" gorethink:"id"`
	Time      int64     `json:"time" gorethink:"time"`
	Body      string    `json:"body" gorethink:"body"`
	Name      string    `json:"name,omitempty" gorethink:"name,omitempty"`
	Trip      string    `json:"trip,omitempty" gorethink:"trip,omitempty"`
	Auth      string    `json:"auth,omitempty" gorethink:"auth,omitempty"`
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
	OP    uint64 `json:"op" gorethink:"op"`
	Board string `json:"board" gorethink:"board"`
}

// DatabasePost is for writing new posts to a database. It contains the IP and
// Password fields, which are never exposed publically through Post.
type DatabasePost struct {
	StandalonePost
	LastUpdated int64    `json:"lastUpdated" gorethink:"lastUpdated"`
	IP          string   `gorethink:"ip"`
	Password    []byte   `gorethink:"password"`
	Log         []string `gorethink:"log"`
}

// LinkMap contains a map of post numbers, this tread is linking, to
// corresponding Link structs
type LinkMap map[uint64]Link

// Link stores the target post's parent board and parent thread
type Link struct {
	OP    uint64 `json:"op" gorethink:"op"`
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
