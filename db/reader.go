package db

import (
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
)

// Preconstructed REQL queries that don't have to be rebuilt
var (
	// Retrieves all threads for the /all/ metaboard
	getAllBoard = r.
			Table("threads").
			EqJoin("id", r.Table("posts")).
			Zip().
			Without(omitForBoards)

	// Fields to omit in board queries. Decreases payload of DB replies.
	omitForBoards = []string{
		"body", "password", "commands", "links", "backlinks", "ip", "editing",
		"op",
	}

	// Fields to omit for post queries
	omitForPosts       = []string{"password", "ip", "lastUpdated"}
	omitForOP          = append(omitForPosts, "op")
	omitForThreadPosts = append(omitForPosts, []string{"op", "board"}...)
)

// GetThread retrieves public thread data from the database
func GetThread(id int64, lastN int) (*types.Thread, error) {
	q := r.
		Table("threads").
		GetAll(id). // Can not join after Get(). Meh.
		EqJoin("id", r.Table("posts")).
		Zip()

	getPosts := r.
		Table("posts").
		GetAllByIndex("op", id).
		OrderBy("id").
		CoerceTo("array")

	// Only fetch last N number of replies
	if lastN != 0 {
		getPosts = getPosts.Slice(-lastN)
	}

	q = q.Merge(map[string]r.Term{
		"posts": getPosts.Without(omitForThreadPosts),
	}).
		Without(omitForOP)

	var thread types.Thread
	if err := One(q, &thread); err != nil {
		return nil, err
	}

	// Remove OP from posts slice to prevent possible duplication. Post might
	// be deleted before the thread due to a deletion race.
	if len(thread.Posts) != 0 && thread.Posts[0].ID == id {
		thread.Posts = thread.Posts[1:]
	}

	return &thread, nil
}

// GetPost reads a single post from the database
func GetPost(id int64) (post types.StandalonePost, err error) {
	q := FindPost(id).Without(omitForPosts).Default(nil)
	err = One(q, &post)
	return
}

// GetBoard retrieves all OPs of a single board
func GetBoard(board string) (*types.Board, error) {
	ctr, err := BoardCounter(board)
	if err != nil {
		return nil, err
	}

	q := r.
		Table("threads").
		GetAllByIndex("board", board).
		EqJoin("id", r.Table("posts")).
		Zip().
		Without(omitForBoards)
	out := &types.Board{Ctr: ctr}
	err = All(q, &out.Threads)

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
