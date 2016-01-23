package server

import (
	"github.com/Soreil/mnemonics"
	r "github.com/dancannon/gorethink"
)

// Reader reads on formats thread and post structs
type Reader struct {
	board                             string
	ident                             Ident
	canSeeMnemonics, canSeeModeration bool
}

// NewReader constructs a new Reader instance
func NewReader(board string, ident Ident) *Reader {
	return &Reader{
		board:            board,
		ident:            ident,
		canSeeMnemonics:  checkAuth("seeMnemonics", ident),
		canSeeModeration: checkAuth("seeModeration", ident),
	}
}

// Used to query equal joins of thread + OP from the DB
type joinedThread struct {
	Left  Thread `gorethink:"left"`
	Right Post   `gorethink:"right"`
}

// GetThread retrieves thread JSON from the database
func (rd *Reader) GetThread(id uint64, lastN int) *ThreadContainer {
	// Verify thread exists. In case of HTTP requests, we kind of do 2
	// validations, but it's better to keep reader uniformity
	if !validateOP(id, rd.board) || !canAccessThread(id, rd.board, rd.ident) {
		return new(ThreadContainer)
	}

	// Keep same format as multiple thread queries
	var thread joinedThread
	db()(r.Object(map[string]r.Term{
		"left":  getThreadMeta(getThread(id)),
		"right": getPost(id),
	})).
		One(&thread)

	// Get all other posts
	var posts []*Post
	query := r.Table("posts").
		GetAllByIndex("op", id).
		Filter(r.Row.Field("id").Eq(id).Not()) // Exclude OP
	if lastN != 0 { // Only fetch last N number of replies
		query = query.Slice(-lastN + 1)
	}
	db()(query).All(&posts)

	// Parse posts, remove those that the client can not access and allocate the
	// rest to a map
	filtered := make(map[string]*Post, len(posts))
	for _, post := range posts {
		if rd.parsePost(post) {
			filtered[idToString(post.ID)] = post
		}
	}

	// Guranteed to have access rights, if thread is accessable
	rd.parsePost(&thread.Right)

	// Compose into the client-side thread type
	return &ThreadContainer{
		Thread: thread.Left,
		Post:   thread.Right,
		Posts:  filtered,
	}
}

// Merges thread counters into the Left field of joinedThread
func getThreadMeta(thread r.Term) r.Term {
	id := thread.Field("id")
	return thread.Merge(map[string]map[string]r.Term{
		"left": map[string]r.Term{
			// Count number of posts
			"postCtr": r.Table("posts").
				GetAllByIndex("op", id).
				Count().
				Sub(1),

			// Image count
			"imageCtr": r.Table("posts").
				GetAllByIndex("op", id).
				HasFields("image").
				Count().
				Sub(1),
		},
	})
}

// parsePost formats the Post struct according to the access level of the
// current client
func (rd *Reader) parsePost(post *Post) bool {
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
		mnem, err := mnemonic.Mnemonic(post.IP)
		throw(err)
		post.Mnemonic = mnem
	}
	return true
}

// GetPost reads a single post from the database
func (rd *Reader) GetPost(id uint64) (post *Post) {
	db()(getPost(id)).One(post)
	rd.parsePost(post)
	return post
}

// GetBoard retrives all OPs of a single board
func (rd *Reader) GetBoard() (board *Board) {
	var threads []*joinedThread
	db()(
		r.Table("threads").
			GetAllByIndex("board", rd.board).
			EqJoin("id", r.Table("posts")).
			ForEach(getThreadMeta),
	).
		All(&threads)
	board.Ctr = boardCounter(rd.board)
	board.Threads = rd.parseThreads(threads)
	return
}

// GetAllBoard retrieves all threads the client has access to for the "/all/"
// meta-board
func (rd *Reader) GetAllBoard() (board *Board) {
	query := r.Table("threads").
		EqJoin("id", r.Table("posts")).
		ForEach(getThreadMeta)

	// Exclude staff board, if no access
	if !canAccessBoard(config.Boards.Staff, rd.ident) {
		query = query.Filter(r.Row.Field("board").Eq(config.Boards.Staff).Not())
	}

	var threads []*joinedThread
	db()(query).All(&threads)
	board.Ctr = postCounter()
	board.Threads = rd.parseThreads(threads)
	return
}

// Parse and format thread query results and discarding those, that the client
// can't access
func (rd *Reader) parseThreads(threads []*joinedThread) []*ThreadContainer {
	filtered := make([]*ThreadContainer, 0, len(threads))
	for _, thread := range threads {
		if thread.Left.Deleted && !rd.canSeeModeration {
			continue
		}
		filtered = append(filtered, &ThreadContainer{
			Thread: thread.Left,
			Post:   thread.Right,
		})
	}
	return filtered
}
