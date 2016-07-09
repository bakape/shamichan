// Package types contains common shared types used throughout the project.
package types

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
)

// Board stores board metadata and the OPs of all threads
type Board struct {
	Ctr     int64    `json:"ctr"`
	Threads []Thread `json:"threads"`
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
	Subject   string         `json:"subject,omitempty" gorethink:"subject"`
	Posts     map[int64]Post `json:"posts,omitempty" gorethink:"posts"`
}

// DatabaseThread is a template for wririting new threads to the database
type DatabaseThread struct {
	PostCtr   int16          `gorethink:"postCtr"`
	ImageCtr  int16          `gorethink:"imageCtr"`
	ID        int64          `gorethink:"id"`
	BumpTime  int64          `gorethink:"bumpTime"`
	ReplyTime int64          `gorethink:"replyTime"`
	Subject   string         `gorethink:"subject,omitempty"`
	Board     string         `gorethink:"board"`
	Posts     map[int64]Post `gorethink:"posts"`
	Log       [][]byte       `gorethink:"log"`
}

// ThreadCreationRequest contains data for creating a thread passed from the
// client theough websockets
type ThreadCreationRequest struct {
	PostCredentials
	Subject string `json:"subject"`
	Board   string `json:"board"`
	Body    string `json:"body"`
}

// PostCredentials contains the common poster credential part of thread and
// reply creation requests
type PostCredentials struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Auth     string `json:"auth"`
	Password string `json:"password"`
}

// Post is a generic post. Either OP or reply.
type Post struct {
	Editing   bool      `json:"editing" gorethink:"editing"`
	OP        int64     `json:"op,omitempty" gorethink:"op"`
	ID        int64     `json:"id" gorethink:"id"`
	Time      int64     `json:"time" gorethink:"time"`
	Board     string    `json:"board" gorethink:"board"`
	IP        string    `json:"-" gorethink:"ip"`
	Password  string    `json:"-" gorethink:"password"`
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

// LinkMap contains a map of post numbers, this tread is linking, to
// corresponding Link structs
type LinkMap map[int64]Link

// Link stores the target post's parent board and parent thread
type Link struct {
	Board string `json:"board" gorethink:"board"`
	OP    int    `json:"op" gorethink:"op"`
}

// Command contains the type and value array of hash commands, such as dice
// rolls, #flip, #8ball, etc. The Val field depends on the Type field.
// Dice: []uint16
// Flip: bool
// EightBall: string
// SyncWatch: TODO: SyncWatch storage type
// Pyu: int64
type Command struct {
	Type CommandType `json:"type" gorethink:"type"`
	Val  interface{} `json:"val" gorethink:"val"`
}
