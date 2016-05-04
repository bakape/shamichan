package db

import (
	"fmt"
	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
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
	err := DB(getThread(id).Merge(toMerge...).Without("log")).One(&thread)
	if err != nil {
		return nil, err
	}

	// Remove OP from posts map to prevent possible duplication
	delete(thread.Posts, util.IDToString(id))

	// Parse posts, remove those that the client can not access and allocate
	// the rest to a map
	filtered := make(map[string]types.Post, len(thread.Posts))
	for _, post := range thread.Posts {
		parsed := rd.parsePost(post)
		if parsed.ID != 0 {
			filtered[util.IDToString(parsed.ID)] = parsed
		}
	}
	thread.Posts = filtered

	return &thread, nil
}

// parsePost formats the Post struct for public distribution
func (rd *Reader) parsePost(post types.Post) types.Post {
	if post.Deleted {
		return types.Post{}
	}
	if post.ImgDeleted {
		post.Image = types.Image{}
		post.ImgDeleted = false
	}
	return post
}

// GetPost reads a single post from the database
func (rd *Reader) GetPost(id, op int64, board string) (
	post types.Post, err error,
) {
	if !auth.CanAccessBoard(board, rd.ident) {
		return types.Post{}, nil
	}

	thread := getThread(op)
	query := r.
		Branch(
			thread.Field("board").Eq(board),
			thread.Field("posts").Field(util.IDToString(id)),
			nil,
		).
		Default(nil)

	err = DB(query).One(&post)
	if err != nil {
		msg := fmt.Sprintf("Error retrieving post: %d", id)
		err = util.WrapError(msg, err)
		return
	}
	if post.ID == 0 {
		return types.Post{}, nil
	}
	return rd.parsePost(post), nil
}

// GetBoard retrieves all OPs of a single board
func (rd *Reader) GetBoard(board string) (out *types.Board, err error) {
	query := r.
		Table("threads").
		GetAllByIndex("board", board).
		Merge(getThreadOP, getLogCounter).
		Without("posts", "log")
	out = &types.Board{}
	err = DB(query).All(&out.Threads)
	if err != nil {
		msg := fmt.Sprintf("Error retrieving board: %s", board)
		err = util.WrapError(msg, err)
		return
	}

	out.Ctr, err = BoardCounter(board)
	if err != nil {
		return
	}

	out.Threads = rd.parseThreads(out.Threads)
	return
}

// GetAllBoard retrieves all threads the client has access to for the "/all/"
// meta-board
func (rd *Reader) GetAllBoard() (board *types.Board, err error) {
	// Can not cast a slice type to another slice type, so create a new slice
	// to pass to GetAllByIndex()
	enabled := config.Get().Boards.Enabled
	args := make([]interface{}, len(enabled))
	for i := 0; i < len(enabled); i++ {
		args[i] = enabled[i]
	}

	query := r.
		Table("threads").
		GetAllByIndex("board", args...).
		Merge(getThreadOP, getLogCounter).
		Without("posts", "log")
	board = &types.Board{}
	err = DB(query).All(&board.Threads)
	if err != nil {
		err = util.WrapError("Error retrieving /all/ board", err)
		return
	}

	board.Ctr, err = PostCounter()
	if err != nil {
		return
	}

	board.Threads = rd.parseThreads(board.Threads)
	return
}

// Parse and format board query results and discard those threads, that the
// client can't access
func (rd *Reader) parseThreads(threads []types.Thread) []types.Thread {
	filtered := make([]types.Thread, 0, len(threads))
	for _, thread := range threads {
		if thread.Deleted {
			continue
		}
		filtered = append(filtered, thread)
	}
	return filtered
}
