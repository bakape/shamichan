package db

import (
	"fmt"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/types"
	"github.com/bakape/meguca/util"
	r "github.com/dancannon/gorethink"
)

// Preconstructed REQL queries that don't have to be rebuilt
var (
	postRow = r.Row.Field("posts")

	// Retrieves thread OP information for merging into the thread struct
	getThreadOP = postRow.Field(r.Row.Field("id").CoerceTo("string"))

	// Retrieves thread counters for merging into the thread struct
	getLogCounter = map[string]r.Term{
		// Replication log counter
		"logCtr": r.Row.Field("log").Count(),
	}
)

// Reader reads on formats thread, post and board structs
type Reader struct {
	ident auth.Ident
}

// NewReader constructs a new Reader instance
func NewReader(ident auth.Ident) *Reader {
	return &Reader{
		ident: ident,
	}
}

// GetThread retrieves thread JSON from the database
func (rd *Reader) GetThread(id int64, lastN int) (*types.Thread, error) {
	toMerge := []interface{}{getThreadOP, getLogCounter}

	// Only fetch last N number of replies
	if lastN != 0 {
		sliced := postRow.
			CoerceTo("array").
			Slice(-lastN).
			CoerceTo("object")
		toMerge = append(toMerge, map[string]r.Term{
			"posts": r.Literal(sliced),
		})
	}

	var thread types.Thread
	err := DB(getThread(id).Merge(toMerge...).Without("log", "op")).One(&thread)
	if err != nil {
		return nil, err
	}

	// Remove OP from posts map to prevent possible duplication
	delete(thread.Posts, util.IDToString(id))

	return &thread, nil
}

// GetPost reads a single post from the database
func (rd *Reader) GetPost(id, op int64) (post types.Post, err error) {
	query := getThread(op).
		Field("posts").
		Field(util.IDToString(id)).
		Default(nil)

	err = DB(query).One(&post)
	if err != nil {
		msg := fmt.Sprintf("Error retrieving post: %d", id)
		err = util.WrapError(msg, err)
	}

	return
}

// GetBoard retrieves all OPs of a single board
func (rd *Reader) GetBoard(board string) (out *types.Board, err error) {
	query := r.
		Table("threads").
		GetAllByIndex("board", board).
		Merge(getThreadOP, getLogCounter).
		Without("posts", "log", "op")
	out = &types.Board{}
	err = DB(query).All(&out.Threads)
	if err != nil {
		msg := fmt.Sprintf("Error retrieving board: %s", board)
		err = util.WrapError(msg, err)
		return
	}

	out.Ctr, err = BoardCounter(board)

	return
}

// GetAllBoard retrieves all threads the client has access to for the "/all/"
// meta-board
func (rd *Reader) GetAllBoard() (board *types.Board, err error) {
	query := r.
		Table("threads").
		Merge(getThreadOP, getLogCounter).
		Without("posts", "log", "op")
	board = &types.Board{}
	err = DB(query).All(&board.Threads)
	if err != nil {
		err = util.WrapError("Error retrieving /all/ board", err)
		return
	}

	board.Ctr, err = PostCounter()

	return
}
