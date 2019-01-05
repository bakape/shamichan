// Package common contains common shared types, variables and constants used
// throughout the project
package common

// ParseBody forwards parser.ParseBody to avoid cyclic imports in db/upkeep
// TODO: Clean up this function signature
var ParseBody func([]byte, string, uint64, uint64, string, bool) ([]Link, []Command, error)

// Board is defined to enable marshalling optimizations and sorting by sticky
// threads
type Board struct {
	Pages   int      `json:"pages"`
	Threads []Thread `json:"threads"`
}

func (b Board) Len() int {
	return len(b.Threads)
}

func (b Board) Swap(i, j int) {
	b.Threads[i], b.Threads[j] = b.Threads[j], b.Threads[i]
}

func (b Board) Less(i, j int) bool {
	// So it gets sorted with sticky threads first
	return b.Threads[i].Sticky
}

// Thread is a transport/export wrapper that stores both the thread metadata,
// its opening post data and its contained posts. The composite type itself is
// not stored in the database.
type Thread struct {
	Abbrev    bool   `json:"abbrev"`
	Sticky    bool   `json:"sticky"`
	Locked    bool   `json:"locked"`
	PostCtr   uint32 `json:"postCtr"`
	ImageCtr  uint32 `json:"imageCtr"`
	ReplyTime int64  `json:"replyTime"`
	BumpTime  int64  `json:"bumpTime"`
	Subject   string `json:"subject"`
	Board     string `json:"board"`
	Post
	Posts []Post `json:"posts"`
}

// Post is a generic post exposed publically through the JSON API. Either OP or
// reply.
type Post struct {
	Editing    bool              `json:"editing"`
	Moderated  bool              `json:"-"`
	Sage       bool              `json:"sage"`
	ID         uint64            `json:"id"`
	Time       int64             `json:"time"`
	Body       string            `json:"body"`
	Flag       string            `json:"flag"`
	PosterID   string            `json:"posterID"`
	Name       string            `json:"name"`
	Trip       string            `json:"trip"`
	Auth       string            `json:"auth"`
	Image      *Image            `json:"image"`
	Links      []Link            `json:"links"`
	Commands   []Command         `json:"commands"`
	Moderation []ModerationEntry `json:"moderaion"`
}

// Return if post has been deleted by staff
func (p *Post) IsDeleted() bool {
	for _, l := range p.Moderation {
		if l.Type == DeletePost {
			return true
		}
	}
	return false
}

// Link describes a link from one post to another
type Link struct {
	ID    uint64 `json:"id"`
	OP    uint64 `json:"op"`
	Board string `json:"board"`
}

// StandalonePost is a post view that includes the "op" and "board" fields,
// which are not exposed though Post, but are required for retrieving a post
// with unknown parenthood.
type StandalonePost struct {
	Post
	OP    uint64 `json:"op"`
	Board string `json:"board"`
}
