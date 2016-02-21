package db

import (
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
func (rd *Reader) GetThread(id uint64, lastN int) types.ThreadContainer {
	thread := getJoinedThread(id)

	// Get all other posts
	var posts []types.Post
	query := r.Table("posts").
		GetAllByIndex("op", id).
		Filter(r.Row.Field("id").Eq(id).Not()) // Exclude OP
	if lastN != 0 { // Only fetch last N number of replies
		query = query.CoerceTo("array").OrderBy("id").Slice(-lastN)
	}
	DB()(query).All(&posts)

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
	}
}

// Retrieve the thread metadata along with the OP post in the same format as
// multiple thread joins, for interoperability
func getJoinedThread(id uint64) (thread joinedThread) {
	DB()(r.
		Expr(map[string]r.Term{
			"left":  getThread(id),
			"right": getPost(id),
		}).
		Merge(getThreadMeta()),
	).One(&thread)
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
func (rd *Reader) GetPost(id uint64) (post types.Post) {
	DB()(getPost(id)).One(&post)
	if post.ID == 0 || !auth.CanAccessBoard(post.Board, rd.ident) {
		return types.Post{}
	}
	var deleted bool // Check if parent thread was not deleted
	DB()(getThread(post.OP).Field("deleted").Default(false)).One(&deleted)
	if deleted {
		return types.Post{}
	}
	return rd.parsePost(post)
}

// GetBoard retrieves all OPs of a single board
func (rd *Reader) GetBoard() (board types.Board) {
	var threads []joinedThread
	DB()(r.
		Table("threads").
		GetAllByIndex("board", rd.board).
		EqJoin("id", r.Table("posts")).
		Merge(getThreadMeta()).
		Without(map[string]string{"right": "op"}),
	).All(&threads)
	board.Ctr = BoardCounter(rd.board)
	board.Threads = rd.parseThreads(threads)
	return
}

// GetAllBoard retrieves all threads the client has access to for the "/all/"
// meta-board
func (rd *Reader) GetAllBoard() (board types.Board) {
	var threads []joinedThread
	DB()(r.Table("threads").
		EqJoin("id", r.Table("posts")).
		Merge(getThreadMeta()).
		Without(map[string]string{"right": "op"}),
	).All(&threads)
	board.Ctr = PostCounter()
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
