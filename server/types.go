/*
 Commonly used types in a single place. Purely for organisation purposes.
*/

package server

// Board stores board metadata and the OPs of all threads
type Board struct {
	Ctr     uint64    `json:"ctr",gorethink:"ctr"`
	Threads []*Thread `json:"threads",gorethink:"threads"`
}

// Thread stores the metadata and posts of a single thread
type Thread struct {
	ID        uint64          `json:"id",gorethink:"id"`
	Time      int64           `json:"time",gorethink:"time"`
	BumpTime  int64           `json:"bumpTime",gorethink:"bumpTime"`
	ReplyTime int64           `json:"replyTime",gorethink:"replyTime"`
	HistCtr   uint64          `json:"histCtr",gorethink:"histCtr"`
	ReplyCtr  uint16          `json:"replyCtr",gorethink:"replyCtr"`
	ImageCtr  uint16          `json:"imageCtr",gorethink:"imageCtr"`
	Locked    bool            `json:"locked,omitempty",gorethink:"locked,omitempty"`
	Archived  bool            `json:"archived,omitempty",gorethink:"archived,omitempty"`
	Sticky    bool            `json:"sticky,omitempty",gorethink:"sticky,omitempty"`
	Nonce     string          `json:"-",gorethink:"nonce"`
	Board     string          `json:"board",gorethink:"board"`
	IP        string          `json:"-",gorethink:"ip"`
	OP        Post            `json:"-",gorethink:"op"` // For internal use
	Posts     map[string]Post `json:"posts,omitempty",gorethink:"posts"`
	History   []Message       `json:"-",gorethink:"history"`
}

// Message is the universal transport container of all live updates through
// websockets
type Message struct {
	Type string `json:"type",gorethink:"type"`

	// If present, determines a priviledged access level, the client has to
	// have, to recieve this message
	Priv string `json:"priv,omitempty",gorethink:"priv,omitempty"`

	// The actual contents of the message. Very variadic, thus interface{}.
	Msg interface{} `json:"msg,omitempty",gorethink:"msg,omitempty"`
}

// Post is a generic post. Either OP or reply.
type Post struct {
	Editing    bool           `json:"editing",gorethink:"editing"`
	Deleted    bool           `json:"-",gorethink:"deleted"`
	ImgDeleted bool           `json:"-",gorethink:"imgDeleted"`
	OP         uint64         `json:"op",gorethink:"op"`
	ID         uint64         `json:"id",gorethink:"id"`
	Time       int64          `json:"time",gorethink:"time"`
	IP         string         `json:"-",gorethink:"ip"`
	Board      string         `json:"board",gorethink:"board"`
	Nonce      string         `json:"-",gorethink:"nonce"`
	Body       string         `json:"body",gorethink:"body"`
	Name       string         `json:"name,omitempty",gorethink:"name,omitempty"`
	Trip       string         `json:"trip,omitempty",gorethink:"trip,omitempty"`
	Auth       string         `json:"auth,omitempty",gorethink:"auth,omitempty"`
	Email      string         `json:"email,omitempty",gorethink:"email,omitempty"`
	Image      *Image         `json:"image,omitempty",gorethink:"image,omitempty"`
	Backlinks  LinkMap        `json:"backlinks,omitempty",gorethink:"backlinks,omitempty"`
	Links      LinkMap        `json:"links,omitempty",gorethink:"links,omitempty"`
	Dice       Dice           `json:"dice,omitempty",gorethink:"dice,omitempty"`
	Mod        ModerationList `json:"mod,omitempty",gorethink:"mod,omitempty"`
}

// Image contains a post's image and thumbanail data
type Image struct {
	APNG    bool      `json:"apng,omitempty",gorethink:"apng,omitempty"`
	Audio   bool      `json:"audio,omitempty",gorethink:"audio,omitempty"`
	Spoiler uint16    `json:"spoiler,omitempty",gorethink:"spoiler,omitempty"`
	Length  [3]uint8  `json:"lenght,omitempty",gorethink:"lenght,omitempty"`
	Dims    [2]uint32 `json:"dims",gorethink:"dims"`
	Size    uint      `json:"size",gorethink:"size"`
	Mid     string    `json:"mid,omitempty",gorethink:"mid,omitempty"`
	Thumb   string    `json:"thumb,omitempty",gorethink:"thumb,omitempty"`
	Src     string    `json:"src",gorethink:"src"`
	Ext     string    `json:"ext",gorethink:"ext"`
	MD5     string
	SHA1    string
	Imgnm   string `json:"imgnm",gorethink:"imgnm"`
}

// Dice stores # command information of the post in exectution order
type Dice []Roll

// Roll represents a single hash command. It always contains the Type field,
// which determines, which of the other fields are present.
type Roll struct {
	Type   string  `json:"type",gorethink:"type"`
	Bool   bool    `json:"bool,omitempty",gorethink:"bool,omitempty"`
	Int    int     `json:"int,omitempty",gorethink:"int,omitempty"`
	Ints   []uint8 `json:"ints,omitempty",gorethink:"ints,omitempty"`
	String string  `json:"string,omitempty",gorethink:"string,omitempty"`
}

// LinkMap contains a map of post numbers, this tread is linking, to
// corresponding Link tuples
type LinkMap map[string]Link

// Link stores the target post's parent board and parent thread
type Link struct {
	Board string `json:"board",gorethink:"board"`
	OP    int    `json:"op",gorethink:"op"`
}

// Ident is used to verify a client's access and write permissions
type Ident struct {
	// Indicates priveledged access rights for staff.
	Auth   string
	IP     string
	Banned bool
}

// ModerationList contains modration acts commited on this post.
// TEMP
type ModerationList []interface{}
