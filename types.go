/*
 Commonly used types in a single place. Purely for organisation purposes.
*/

package main

// Thread stores the metadata and posts of a single thread
type Thread struct {
	ID       int             `json:"id",gorethink:"id"`
	IP       string          `json:"ip",gorethink:"ip"`
	Board    string          `json:"board",gorethink:"board"`
	Time     int             `json:"time",gorethink:"time"`
	BumpTime int             `json:"bumpTime",gorethink:"bumpTime"`
	Nonce    string          `json:"nonce",gorethink:"nonce"`
	Posts    map[string]Post `json:"posts",gorethink:"posts"`
	History  []Message       `json:"history",gorethink:"history"`
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
	ID         int     `json:"id",gorethink:"id"`
	IP         string  `json:"ip",gorethink:"ip"`
	OP         int     `json:"op",gorethink:"op"`
	Time       int     `json:"time",gorethink:"time"`
	Nonce      string  `json:"nonce",gorethink:"nonce"`
	Editing    bool    `json:"editing,omitempty",gorethink:"editing,omitempty"`
	Body       string  `json:"body",gorethink:"body"`
	Deleted    bool    `json:"deleted",gorethink:"deleted"`
	ImgDeleted bool    `json:"imgDeleted",gorethink:"imgDeleted"`
	Image      Image   `json:"image,omitempty",gorethink:"image,omitempty"`
	Name       string  `json:"name,omitempty",gorethink:"name,omitempty"`
	Trip       string  `json:"trip,omitEmpty",gorethink:"trip,omitEmpty"`
	Email      string  `json:"email,omitempty",gorethink:"email,omitempty"`
	Auth       string  `json:"auth,omitempty",gorethink:"auth,omitempty"`
	Dice       Dice    `json:"dice,omitempty",gorethink:"dice,omitempty"`
	Links      LinkMap `json:"links,omitempty",gorethink:"links,omitempty"`
	Backlinks  LinkMap `json:"backlinks,omitempty",gorethink:"backlinks,omitempty"`
}

// Image contains a post's image and thumbanail data
type Image struct {
	Src     string `json:"src",gorethink:"src"`
	Thumb   string `json:"thumb,omitempty",gorethink:"thumb,omitempty"`
	Mid     string `json:"mid,omitempty",gorethink:"mid,omitempty"`
	Dims    [2]int `json:"dims",gorethink:"dims"`
	Ext     string `json:"ext",gorethink:"ext"`
	Size    int    `json:"size",gorethink:"size"`
	MD5     string
	SHA1    string
	Imgnm   string `json:"imagnm",gorethink:"imagnm"`
	Spoiler int    `json:"spoiler,omitempty",gorethink:"spoiler,omitempty"`
	APNG    bool   `json:"apng,omitempty",gorethink:"apng,omitempty"`
	Audio   bool   `json:"audio,omitempty",gorethink:"audio,omitempty"`
	Length  string `json:"lenght,omitempty",gorethink:"lenght,omitempty"`
}

// Dice stores # command information of the post in exectution order
type Dice []Roll

// Roll represents a single hash command. It always contains the Type field,
// which determines, which of the other fields are present.
type Roll struct {
	Type   string `json:"type",gorethink:"type"`
	Bool   bool   `json:"bool,omitempty",gorethink:"bool,omitempty"`
	Int    int    `json:"int,omitempty",gorethink:"int,omitempty"`
	Ints   []int  `json:"ints,omitempty",gorethink:"ints,omitempty"`
	String string `json:"string,omitempty",gorethink:"string,omitempty"`
}

// LinkMap contains a map of post numbers, this tread is linking, to
// corresponding Link tuples
type LinkMap map[string]Link

// Link stores the target post's parent board and parent thread
type Link struct {
	Board string `json:"board",gorethink:"board"`
	ID    int    `json:"id",gorethink:"id"`
}

// Ident is used to verify a client's access and write permissions
type Ident struct {
	// Indicates priveledged access rights for staff.
	Auth string
	Ban  bool
	IP   string
}
