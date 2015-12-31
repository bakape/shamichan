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
func (rd *Reader) GetThread(id, lastN int) *Thread {
	// Verify thread exists
	if !validateOP(id, rd.board) {
		return new(Thread)
	}
	res := rd.threadQuery(getThread(id))

	// Only show the last N post
	if lastN != 0 {
		res = res.Merge(termMap{
			"posts": res.Field("posts").
				CoerceTo("array").
				Slice(-lastN + 1).
				CoerceTo("object"),
		})
	}
	thread := new(Thread)
	rGet(res).One(thread)

	// Verify thread access rights
	if !rd.parsePost(&thread.OP) {
		return new(Thread)
	}

	// Place the retrieved OP into the Posts map and override duplicate, if any
	thread.Posts[strconv.Itoa(thread.ID)] = thread.OP

	for id, post := range thread.Posts {
		if !rd.parsePost(&post) {
			delete(thread.Posts, id)
		}
	}
	return thread
}

// threadQuery constructs the common part of a all thread queries
func (rd *Reader) threadQuery(thread r.Term) r.Term {
	return thread.Merge(termMap{
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
			post.Image = &Image{}
		}
		post.Mod = ModerationList{}
	}
	if rd.canSeeMnemonics {
		// Mnemonic generation call
	}
	return true
}

// GetPost reads a single post from the database
func (rd *Reader) GetPost(id int) *Post {
	op := parentThread(id)
	post := new(Post)
	if op == 0 { // Post does not exist
		return post
	}
	rGet(getPost(id, op)).One(post)
	rd.parsePost(post)
	return post
}

// GetBoard retrives all OPs of a single board
func (rd *Reader) GetBoard() *Board {
	board := new(Board)
	rGet(r.Table("threads").
		GetAllByIndex("board", rd.board).
		ForEach(rd.threadQuery).
		Without("posts"),
	).
		All(&board.Threads)
	board.Ctr = boardCounter(rd.board)
	board.Threads = rd.filterThreads(board.Threads)
	return board
}

// GetAllBoard retrieves all threads the client has access to for the "/all/"
// meta-board
func (rd *Reader) GetAllBoard() *Board {
	query := r.Table("threads")

	// Exclude staff board, if no access
	if !canAccessBoard(config.Boards.Staff, rd.ident) {
		query = query.Filter(func(thread r.Term) r.Term {
			return thread.Field("board").Eq(config.Boards.Staff).Not()
		})
	}

	board := new(Board)
	rGet(query.ForEach(rd.threadQuery).Without("posts")).All(&board.Threads)
	board.Ctr = postCounter()
	board.Threads = rd.filterThreads(board.Threads)
	return board
}

// Filter a slice of thread pointers by parsing and formating their OPs and
// discarding those, that the client can't access.
func (rd *Reader) filterThreads(threads []*Thread) []*Thread {
	filtered := make([]*Thread, 0, len(threads))
	for _, thread := range threads {
		if rd.parsePost(&thread.OP) {
			// Mimics structure of regular threads, for uniformity
			thread.Posts = map[string]Post{
				strconv.Itoa(thread.ID): thread.OP,
			}
			filtered = append(filtered, thread)
		}
	}
	return filtered
}
