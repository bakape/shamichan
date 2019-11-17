// Package common contains common shared types, variables and constants used
// throughout the project
package common

// Contains a specific page of the thread index.
//
// A thread index can either contain all threads on the page or filtered by
// specific tags.
type ThreadIndex struct {
	Page    int      `json:"page"`
	Pages   int      `json:"pages"`
	Threads []Thread `json:"threads"`
}

// Stores thread meta-data and contained posts
type Thread struct {
	Locked     bool   `json:"locked"`
	PostCount  uint32 `json:"post_count"`
	ImageCount uint32 `json:"image_count"`
	Page       uint64 `json:"page"`
	ID         uint64 `json:"id"`
	UpdateTime int64  `json:"update_time"`
	BumpTime   int64  `json:"bump_time"`
	Subject    string `json:"subject"`
	Posts      []Post `json:"posts"`
}

// Post is a generic post exposed publically through the JSON API. Either OP or
// reply.
type Post struct {
	Editing bool   `json:"editing,omitempty"`
	Sage    bool   `json:"sage,omitempty"`
	ID      uint64 `json:"id"`
	Page    uint64 `json:"page"`
	Time    int64  `json:"time"`
	Flag    string `json:"flag,omitempty"`
	Name    string `json:"name,omitempty"`
	Trip    string `json:"trip,omitempty"`
	Body    []byte `json:"body"`
	Image   *Image `json:"image,omitempty"`
}

// StandalonePost is a post view that includes the "op" field, which are not
// exposed though Post, but are required for retrieving a post  with unknown
// parenthood.
type StandalonePost struct {
	Post
	OP uint64 `json:"op"`
}
