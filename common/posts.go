// Package common contains common shared types, variables and constants used
// throughout the project
package common

// Contains a specific page of the board index
type Board struct {
	Page    int      `json:"page"`
	Pages   int      `json:"pages"`
	Threads []Thread `json:"threads"`
}

// Stores thread meta-data and contained posts
type Thread struct {
	Sticky     bool   `json:"sticky"`
	Locked     bool   `json:"locked"`
	PostCount  uint32 `json:"post_count"`
	ImageCount uint32 `json:"image_count"`
	Page       uint32 `json:"page"`
	ID         uint64 `json:"id"`
	UpdateTime int64  `json:"update_time"`
	BumpTime   int64  `json:"bump_time"`
	Subject    string `json:"subject"`
	Board      string `json:"board"`
	Posts      []Post `json:"posts"`
}

// Post is a generic post exposed publically through the JSON API. Either OP or
// reply.
type Post struct {
	Editing    bool              `json:"editing"`
	Sage       bool              `json:"sage"`
	Auth       ModerationLevel   `json:"auth"`
	Page       uint32            `json:"page"`
	ID         uint64            `json:"id"`
	Time       int64             `json:"time"`
	Body       string            `json:"body"`
	Flag       string            `json:"flag"`
	Name       string            `json:"name"`
	Trip       string            `json:"trip"`
	Image      *Image            `json:"image"`
	Links      map[uint64]Link   `json:"links"`
	Commands   []Command         `json:"commands"`
	Moderation []ModerationEntry `json:"moderation"`
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

// Link describes the target post of one post linking another
type Link struct {
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
