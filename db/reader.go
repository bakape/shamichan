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
		"log", "body", "password", "commands", "links", "backlinks", "ip",
		"editing", "op",
	}

	// Fields to omit for post queries
	omitForPosts = []string{"log", "password", "ip"}
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
		"logCtr": r.Row.Field("log").Count(),
		"posts": getPosts.
			Merge(func(post r.Term) map[string]r.Term {
				return map[string]r.Term{
					"logCtr": post.Field("log").Count(),
				}
			}).
			Without(omitForPosts),
	}).
		Without(omitForPosts)

	var thread types.Thread
	if err := One(q, &thread); err != nil {
		return nil, err
	}

	// Remove OP from posts slice to prevent possible duplication
	if thread.Posts[0].ID == id {
		thread.Posts = thread.Posts[1:]
	}

	// Do not include redundant "op" field in output JSON
	thread.OP = 0

	return &thread, nil
}

// GetPost reads a single post from the database
func GetPost(id int64) (post types.Post, err error) {
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
