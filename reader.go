package main

import (
	r "github.com/dancannon/gorethink"
)

// Reader reads on formats thread and post structs
type Reader struct {
	board                             string
	ident                             Ident
	canSeeMnemonics, canSeeModeration bool
}

// Newreader constructs a new Reader struct
func Newreader(board string, ident Ident) *Reader {
	return &Reader{
		board,
		ident,

		// DJs  can see mnemonics, but not commited moderation
		ident.Auth == "dj" || checkAuth("moderator", ident),
		checkAuth("janitor", ident),
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
