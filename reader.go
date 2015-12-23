package main

import (
	r "github.com/dancannon/gorethink"
	"strconv"
)

// Reader reads on formats thread and post structs
type Reader struct {
	board                             string
	ident                             Ident
	canSeeMnemonics, canSeeModeration bool
}

// NewReader constructs a new Reader struct
func NewReader(board string, ident Ident) *Reader {
	return &Reader{
		board:            board,
		ident:            ident,
		canSeeMnemonics:  checkAuth("seeMnemonics", ident),
		canSeeModeration: checkAuth("seeModeration", ident),
	}
}

// GetThread retrieves thread JSON from the database
func (rd *Reader) GetThread(id, lastN int) (thread Thread) {
	// Verify thread exists
	if !validateOP(id, rd.board) {
		return
	}
	res := rd.threadQuery(getThread(id))

	// Only show the last N post
	if lastN != 0 {
		res = res.Merge(updateMap{
			"posts": res.Field("posts").
				CoerceTo("array").
				Slice(-lastN + 1).
				CoerceTo("object"),
		})
	}
	rGet(res).One(&thread)

	// Verify thread access rights
	if !rd.parsePost(&thread.OP) {
		return Thread{}
	}

	// Place the retrieved OP into the Posts map and override duplicate, if any
	thread.Posts[strconv.Itoa(thread.ID)] = thread.OP

	for id, post := range thread.Posts {
		if !rd.parsePost(&post) {
			delete(thread.Posts, id)
		}
	}
	return
}

// threadQuery constructs the common part of a all thread queries
func (rd *Reader) threadQuery(thread r.Term) r.Term {
	return thread.Merge(updateMap{
		// Ensure we always get the OP
		"op": thread.Field("posts").
			Field(thread.Field("id").CoerceTo("string")),
	}).
		Without("history")
}

// parsePost formats the Post struct according to the access level of the
// current client
func (rd *Reader) parsePost(post *Post) bool {
	if post.ID == 0 {
		return false
	}
	if !rd.canSeeModeration {
		if post.Deleted {
			return false
		}
		if post.ImgDeleted {
			post.Image = Image{}
		}
		post.Mod = ModerationList{}
	}
	if rd.canSeeMnemonics {
		// Mnemonic generation call
	}
	return true
}

// GetPost reads a single post from the database
func (rd *Reader) GetPost(id int) (post Post) {
	op := parentThread(id)

	// Post does not exist
	if op == 0 {
		return
	}
	rGet(getPost(id, op)).One(&post)
	rd.parsePost(&post)
	return
}

// GetBoard retrives all OPs of a single board
func (rd *Reader) GetBoard() (board Board) {
	rGet(r.Table("threads").
		GetAllByIndex("board", rd.board).
		ForEach(rd.threadQuery).
		Without("posts"),
	).
		All(&board.Threads)
	rGet(r.Table("main").
		Get("histCounts").
		Field(rd.board).
		Default(0),
	).
		One(&board.Ctr)

	filtered := []Thread{}
	for _, thread := range board.Threads {
		if rd.parsePost(&thread.OP) {
			// Mimics structure of regular threads, for uniformity
			thread.Posts = map[string]Post{
				strconv.Itoa(thread.ID): thread.OP,
			}
			filtered = append(filtered, thread)
		}
	}
	board.Threads = filtered
	return
}
