package db

import (
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

	// Retrieves all threads for the /all/ metaboard
	getAllBoard = r.
			Table("threads").
			Merge(getThreadOP, getLogCounter).
			Without("posts", "log")
)

// GetThread retrieves public thread data from the database
func GetThread(id int64, lastN int) (*types.Thread, error) {
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
	err := One(getThread(id).Merge(toMerge...).Without("log"), &thread)
	if err != nil {
		return nil, err
	}

	// Remove OP from posts map to prevent possible duplication
	delete(thread.Posts, id)

	return &thread, nil
}

// GetPost reads a single post from the database complete with parent board and
// thread
func GetPost(id int64) (post types.StandalonePost, err error) {
	q := FindParentThread(id).
		Do(func(t r.Term) r.Term {
			return t.
				Field("posts").
				Field(util.IDToString(id)).
				Merge(map[string]r.Term{
					"op":    t.Field("id"),
					"board": t.Field("board"),
				})
		}).
		Default(nil)
	err = One(q, &post)
	return
}

// GetBoard retrieves all OPs of a single board
func GetBoard(board string) (*types.Board, error) {
	ctr, err := BoardCounter(board)
	if err != nil {
		return nil, err
	}

	query := r.
		Table("threads").
		GetAllByIndex("board", board).
		Merge(getThreadOP, getLogCounter).
		Without("posts", "log")
	out := &types.Board{Ctr: ctr}
	err = All(query, &out.Threads)

	return out, err
}

// GetAllBoard retrieves all threads for the "/all/" meta-board
func GetAllBoard() (board *types.Board, err error) {
	ctr, err := PostCounter()
	if err != nil {
		return
	}
	board = &types.Board{Ctr: ctr}
	err = All(getAllBoard, &board.Threads)
	return
}
