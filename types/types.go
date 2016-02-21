// Package types contains common shared types used throughout the project.
package types

// Board stores board metadata and the OPs of all threads
type Board struct {
	Ctr     uint64            `json:"ctr"`
	Threads []ThreadContainer `json:"threads,omitempty"`
}

// ThreadContainer is a transport/export wrapper that stores both the thread
// metada, its opening post data and its contained posts. The composite type
// itself is not stored in the database.
type ThreadContainer struct {
	Thread
	Post
	Posts map[string]Post `json:"posts,omitempty"`
}

// Thread stores thread metadata
type Thread struct {
	Locked    bool   `json:"locked,omitempty" gorethink:"locked,omitempty"`
	Archived  bool   `json:"archived,omitempty" gorethink:"archived,omitempty"`
	Sticky    bool   `json:"sticky,omitempty" gorethink:"sticky,omitempty"`
	Deleted   bool   `json:"deleted,omitempty" gorethink:"deleted,omitempty"`
	PostCtr   uint16 `json:"postCtr" gorethink:"postCtr"`
	ImageCtr  uint16 `json:"imageCtr" gorethink:"imageCtr"`
	ID        uint64 `json:"id" gorethink:"id"`
	BumpTime  int64  `json:"bumpTime" gorethink:"bumpTime"`
	ReplyTime int64  `json:"replyTime" gorethink:"replyTime"`
	Board     string `json:"board" gorethink:"board"`
}

// Post is a generic post. Either OP or reply.
type Post struct {
	Editing    bool `json:"editing" gorethink:"editing"`
	Deleted    bool `json:"-" gorethink:"deleted,omitempty"`
	ImgDeleted bool `json:"-" gorethink:"imgDeleted,omitempty"`
	Image
	OP        uint64  `json:"op,omitempty" gorethink:"op"`
	ID        uint64  `json:"id" gorethink:"id"`
	Time      int64   `json:"time" gorethink:"time"`
	IP        string  `json:"-" gorethink:"ip"`
	Board     string  `json:"board" gorethink:"board"`
	Nonce     string  `json:"-" gorethink:"nonce"`
	Body      string  `json:"body" gorethink:"body"`
	Name      string  `json:"name,omitempty" gorethink:"name,omitempty"`
	Trip      string  `json:"trip,omitempty" gorethink:"trip,omitempty"`
	Auth      string  `json:"auth,omitempty" gorethink:"auth,omitempty"`
	Email     string  `json:"email,omitempty" gorethink:"email,omitempty"`
	Backlinks LinkMap `json:"backlinks,omitempty" gorethink:"backlinks,omitempty"`
	Links     LinkMap `json:"links,omitempty" gorethink:"links,omitempty"`
}

// Image contains a post's image and thumbnail data
type Image struct {
	APNG    bool     `json:"apng,omitempty" gorethink:"apng,omitempty"`
	Audio   bool     `json:"audio,omitempty" gorethink:"audio,omitempty"`
	Spoiler uint8    `json:"spoiler,omitempty" gorethink:"spoiler,omitempty"`
	Ext     uint8    `json:"ext,omitempty" gorethink:"ext,omitempty"`
	Length  int32    `json:"length,omitempty" gorethink:"length,omitempty"`
	Dims    []uint16 `json:"dims,omitempty" gorethink:"dims,omitempty"`
	File    string   `json:"file,omitempty" gorethink:"file,omitempty"`
	Size    int64    `json:"size,omitempty" gorethink:"size,omitempty"`
	MD5     string   `json:",omitempty" gorethink:",omitempty"`
	SHA1    string   `json:",omitempty" gorethink:",omitempty"`
	Imgnm   string   `json:"imgnm,omitempty" gorethink:"imgnm,omitempty"`
}

// LinkMap contains a map of post numbers, this tread is linking, to
// corresponding Link structs
type LinkMap map[string]Link

// Link stores the target post's parent board and parent thread
type Link struct {
	Board string `json:"board" gorethink:"board"`
	OP    int    `json:"op" gorethink:"op"`
}
