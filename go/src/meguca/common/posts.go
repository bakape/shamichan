//go:generate easyjson --all --no_std_marshalers $GOFILE

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

// ParseBody forwards parser.ParseBody to avoid cyclic imports in db/upkeep
var ParseBody func([]byte, string) ([][2]uint64, []Command, error)

//easyjson:json
// Board is a slice of threads. Defined to enable marshalling optimizations.
type Board []Thread

// Thread is a transport/export wrapper that stores both the thread metadata,
// its opening post data and its contained posts. The composite type itself is
// not stored in the database.
type Thread struct {
	Abbrev    bool   `json:"abbrev,omitempty"`
	Locked    bool   `json:"locked,omitempty"`
	PostCtr   uint32 `json:"postCtr"`
	ImageCtr  uint32 `json:"imageCtr"`
	ReplyTime int64  `json:"replyTime"`
	BumpTime  int64  `json:"bumpTime"`
	LogCtr    uint64 `json:"logCtr"`
	Subject   string `json:"subject"`
	Board     string `json:"board"`
	Post
	Posts []Post `json:"posts"`
}

// Post is a generic post exposed publically through the JSON API. Either OP or
// reply.
type Post struct {
	Editing   bool        `json:"editing,omitempty"`
	Banned    bool        `json:"banned,omitempty"`
	ID        uint64      `json:"id"`
	Time      int64       `json:"time"`
	Body      string      `json:"body"`
	Name      string      `json:"name,omitempty"`
	Trip      string      `json:"trip,omitempty"`
	Auth      string      `json:"auth,omitempty"`
	Links     [][2]uint64 `json:"links,omitempty"`
	Backlinks [][2]uint64 `json:"backlinks,omitempty"`
	Commands  []Command   `json:"commands,omitempty"`
	Image     *Image      `json:"image,omitempty"`
}

// StandalonePost is a post view that includes the "op" and "board" fields,
// which are not exposed though Post, but are required for retrieving a post
// with unknown parenthood.
type StandalonePost struct {
	Post
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
