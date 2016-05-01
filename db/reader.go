package db

import (
	"fmt"
	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/types"
	"github.com/bakape/meguca/util"
	r "github.com/dancannon/gorethink"
)

// Reader reads on formats thread, post and board structs
type Reader struct {
	board string
	ident auth.Ident
}

// NewReader constructs a new Reader instance
func NewReader(board string, ident auth.Ident) *Reader {
	return &Reader{
		board: board,
		ident: ident,
	}
}

// Used to query equal joins of thread + OP from the DB
type joinedThread struct {
	Left  types.Thread `gorethink:"left"`
	Right types.Post   `gorethink:"right"`
}

// GetThread retrieves thread JSON from the database
func (rd *Reader) GetThread(id int64, lastN int) (
	types.ThreadContainer,
	error,
) {
	thread, err := getJoinedThread(id)
	if err != nil {
		return types.ThreadContainer{}, err
	}

	// Get all other posts
	var posts []types.Post
	query := r.Table("posts").
		GetAllByIndex("op", id).
		Filter(r.Row.Field("id").Eq(id).Not()) // Exclude OP
	if lastN != 0 { // Only fetch last N number of replies
		query = query.CoerceTo("array").OrderBy("id").Slice(-lastN)
	}
	err = DB(query).All(&posts)
	if err != nil {
		msg := fmt.Sprintf("Error retrieving thread: %d:last%d", id, lastN)
		return types.ThreadContainer{}, util.WrapError(msg, err)
	}

	// Parse posts, remove those that the client can not access and allocate the
	// rest to a map
	filtered := make(map[string]types.Post, len(posts))
	for _, post := range posts {
		parsed := rd.parsePost(post)
		if parsed.ID != 0 {
			filtered[util.IDToString(parsed.ID)] = parsed
		}
	}

	// No replies in thread or all replies deleted
	if len(filtered) == 0 {
		filtered = map[string]types.Post(nil)
	}

	// Compose into the client-side thread type
	return types.ThreadContainer{
		// Guranteed to have access rights, if thread is accessable
		Post:   rd.parsePost(thread.Right),
		Thread: thread.Left,
		Posts:  filtered,
	}, nil
}

// Retrieve the thread metadata along with the OP post in the same format as
// multiple thread joins, for interoperability
func getJoinedThread(id int64) (thread joinedThread, err error) {
	query := r.
		Expr(map[string]r.Term{
			"left":  getThread(id).Without("log"),
			"right": getPost(id),
		}).
		Merge(getThreadMeta())
	err = DB(query).One(&thread)
	if err != nil {
		msg := fmt.Sprintf("Error retrieving joined thread: %d", id)
		err = util.WrapError(msg, err)
		return
	}
	thread.Right.OP = 0
	return
}

// Merges thread counters into the Left field of joinedThread
func getThreadMeta() map[string]map[string]r.Term {
	id := r.Row.Field("left").Field("id")
	return map[string]map[string]r.Term{
		"left": {
			// Count number of posts
			"postCtr": r.Table("posts").
				GetAllByIndex("op", id).
				Count().
				Sub(1),

			// Image count
			"imageCtr": r.Table("posts").
				GetAllByIndex("op", id).
				HasFields("file").
				Count().
				Sub(1),
		},
	}
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
	post.IP = "" // Never pass IPs client-side
	return post
}

// GetPost reads a single post from the database
func (rd *Reader) GetPost(id int64) (post types.Post, err error) {
	err = DB(getPost(id)).One(&post)
	if err != nil {
		msg := fmt.Sprintf("Error retrieving post: %d", id)
		err = util.WrapError(msg, err)
		return
	}
	if post.ID == 0 || !auth.CanAccessBoard(post.Board, rd.ident) {
		return types.Post{}, nil
	}

	// Check if parent thread was not deleted
	var deleted bool
	err = DB(getThread(post.OP).Field("deleted").Default(false)).One(&deleted)
	if err != nil {
		msg := fmt.Sprintf(
			"Error checking, if parent thread is deleted: %d",
			id,
		)
		return types.Post{}, util.WrapError(msg, err)
	}
	if deleted {
		return types.Post{}, nil
	}
	return rd.parsePost(post), nil
}

// GetBoard retrieves all OPs of a single board
func (rd *Reader) GetBoard() (board types.Board, err error) {
	var threads []joinedThread
	err = DB(r.
		Table("threads").
		GetAllByIndex("board", rd.board).
		EqJoin("id", r.Table("posts")).
		Merge(getThreadMeta()).
		Without(map[string]string{"right": "op"}),
	).
		All(&threads)
	if err != nil {
		err = util.WrapError(
			fmt.Sprintf("Error retrieving board: %s", rd.board),
			err,
		)
		return
	}
	board.Ctr, err = BoardCounter(rd.board)
	if err != nil {
		return
	}
	board.Threads = rd.parseThreads(threads)
	return
}

// GetAllBoard retrieves all threads the client has access to for the "/all/"
// meta-board
func (rd *Reader) GetAllBoard() (board types.Board, err error) {
	var threads []joinedThread
	err = DB(r.Table("threads").
		EqJoin("id", r.Table("posts")).
		Merge(getThreadMeta()).
		Without(map[string]string{"right": "op"}),
	).
		All(&threads)
	if err != nil {
		err = util.WrapError("Error retrieving /all/ board", err)
		return
	}
	board.Ctr, err = PostCounter()
	if err != nil {
		return
	}
	board.Threads = rd.parseThreads(threads)
	return
}

// Parse and format board query results and discard those threads, that the
// client can't access
func (rd *Reader) parseThreads(threads []joinedThread) []types.ThreadContainer {
	filtered := make([]types.ThreadContainer, 0, len(threads))
	for _, thread := range threads {
		if thread.Left.Deleted {
			continue
		}
		filtered = append(filtered, types.ThreadContainer{
			Thread: thread.Left,
			Post:   thread.Right,
		})
	}
	if len(filtered) == 0 {
		filtered = []types.ThreadContainer(nil)
	}
	return filtered
}
