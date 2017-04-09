package db

import (
	"database/sql"
	"meguca/common"
	"meguca/util"
	"sort"

	"github.com/boltdb/bolt"
	"github.com/lib/pq"
)

type imageScanner struct {
	APNG, Audio, Video, Spoiler       sql.NullBool
	FileType, ThumbType, Length, Size sql.NullInt64
	Name, SHA1, MD5                   sql.NullString
	Dims                              pq.Int64Array
}

// Returns and array of pointers to the struct fields for passing to
// rowScanner.Scan()
func (i *imageScanner) ScanArgs() []interface{} {
	return []interface{}{
		&i.APNG, &i.Audio, &i.Video, &i.FileType, &i.ThumbType, &i.Dims,
		&i.Length, &i.Size, &i.MD5, &i.SHA1,
	}
}

// Returns the scanned *common.Image or nil, if none
func (i *imageScanner) Val() *common.Image {
	if !i.SHA1.Valid {
		return nil
	}

	var dims [4]uint16
	for j := range dims {
		dims[j] = uint16(i.Dims[j])
	}

	return &common.Image{
		Spoiler: i.Spoiler.Bool,
		ImageCommon: common.ImageCommon{
			APNG:      i.APNG.Bool,
			Audio:     i.Audio.Bool,
			Video:     i.Video.Bool,
			FileType:  uint8(i.FileType.Int64),
			ThumbType: uint8(i.ThumbType.Int64),
			Length:    uint32(i.Length.Int64),
			Dims:      dims,
			Size:      int(i.Size.Int64),
			MD5:       i.MD5.String,
			SHA1:      i.SHA1.String,
		},
		Name: i.Name.String,
	}
}

type postScanner struct {
	common.Post
	banned, spoiler, deleted    sql.NullBool
	name, trip, auth, imageName sql.NullString
	links, backlinks            linkRow
	commands                    commandRow
}

func (p *postScanner) ScanArgs() []interface{} {
	return []interface{}{
		&p.Editing, &p.banned, &p.spoiler, &p.deleted, &p.ID, &p.Time, &p.Body,
		&p.name, &p.trip, &p.auth, &p.links, &p.backlinks, &p.commands,
		&p.imageName,
	}
}

func (p postScanner) Val() (common.Post, error) {
	p.Banned = p.banned.Bool
	p.Deleted = p.deleted.Bool
	p.Name = p.name.String
	p.Trip = p.trip.String
	p.Auth = p.auth.String
	p.Links = [][2]uint64(p.links)
	p.Backlinks = [][2]uint64(p.backlinks)
	p.Commands = []common.Command(p.commands)

	return p.Post, nil
}

// Returns if image is spoiled and it's assigned name, if any
func (p postScanner) Image() (bool, string) {
	return p.spoiler.Bool, p.imageName.String
}

type postSorter []*common.Post

func (p postSorter) Len() int {
	return len(p)
}

func (p postSorter) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p postSorter) Less(i, j int) bool {
	return p[i].ID < p[j].ID
}

// PostStats contains post open status, body and creation time
type PostStats struct {
	Editing, HasImage bool
	ID                uint64
	Time              int64
	Body              []byte
}

// GetThread retrieves public thread data from the database
func GetThread(id uint64, lastN int) (t common.Thread, err error) {
	tx, err := db.Begin()
	if err != nil {
		return
	}
	defer tx.Commit()
	err = setReadOnly(tx)
	if err != nil {
		return
	}

	// Get thread metadata
	row := tx.Stmt(prepared["get_thread"]).QueryRow(id)
	err = row.Scan(
		&t.Board, &t.PostCtr, &t.ImageCtr, &t.ReplyTime, &t.BumpTime,
		&t.Subject,
	)
	if err != nil {
		return
	}
	t.Abbrev = lastN != 0

	// Get OP post. Need to fetch separately, in case not fetching the full
	// thread. Also allows to return early on deleted threads.
	row = tx.Stmt(prepared["get_thread_post"]).QueryRow(id)
	t.Post, err = scanThreadPost(row)
	if err != nil {
		return
	}

	// Get replies
	var (
		r   *sql.Rows
		cap int
	)
	if lastN == 0 {
		r, err = tx.Stmt(prepared["get_full_thread"]).Query(id)
		cap = int(t.PostCtr)
	} else {
		r, err = tx.Stmt(prepared["get_last_n"]).Query(id, lastN)
		cap = lastN
	}
	if err != nil {
		return
	}
	defer r.Close()
	t.Posts = make([]common.Post, 0, cap)

	var p common.Post
	for r.Next() {
		p, err = scanThreadPost(r)
		if err != nil {
			return
		}
		t.Posts = append(t.Posts, p)
	}
	err = r.Err()
	if err != nil {
		return
	}

	// Inject bodies into open posts
	open := make([]*common.Post, 0, 16)
	if t.Post.Editing {
		open = append(open, &t.Post)
	}
	for i := range t.Posts {
		if t.Posts[i].Editing {
			open = append(open, &t.Posts[i])
		}
	}
	err = injectOpenBodies(open)

	return
}

func scanThreadPost(rs rowScanner) (res common.Post, err error) {
	var (
		args = make([]interface{}, 0, 22)
		post postScanner
		img  imageScanner
	)
	args = append(args, post.ScanArgs()...)
	args = append(args, img.ScanArgs()...)

	err = rs.Scan(args...)
	if err != nil {
		return
	}
	res, err = post.Val()
	if err != nil {
		return
	}
	res.Image = img.Val()
	if res.Image != nil {
		res.Image.Spoiler, res.Image.Name = post.Image()
	}
	return
}

// GetPost reads a single post from the database
func GetPost(id uint64) (res common.StandalonePost, err error) {
	var (
		args = make([]interface{}, 2, 26)
		post postScanner
		img  imageScanner
	)
	args[0] = &res.OP
	args[1] = &res.Board
	args = append(args, post.ScanArgs()...)
	args = append(args, img.ScanArgs()...)

	err = prepared["get_post"].QueryRow(id).Scan(args...)
	if err != nil {
		return
	}
	res.Post, err = post.Val()
	if err != nil {
		return
	}
	res.Image = img.Val()
	if res.Image != nil {
		res.Image.Spoiler, res.Image.Name = post.Image()
	}

	if res.Editing {
		res.Body, err = GetOpenBody(res.ID)
		if err != nil {
			return
		}
	}

	return
}

// GetBoardCatalog retrieves all OPs of a single board
func GetBoardCatalog(board string) (common.Board, error) {
	r, err := prepared["get_board"].Query(board)
	if err != nil {
		return nil, err
	}
	return scanCatalog(r)
}

// GetBoard retrieves all threads on the board, complete with the first 5
// posts
func GetBoard(board string) (common.Board, error) {
	r, err := prepared["get_board_thread_ids"].Query(board)
	if err != nil {
		return nil, err
	}
	return scanBoard(r)
}

// GetAllBoardCatalog retrieves all threads for the "/all/" meta-board
func GetAllBoardCatalog() (common.Board, error) {
	r, err := prepared["get_all_board"].Query()
	if err != nil {
		return nil, err
	}
	return scanCatalog(r)
}

// GetAllBoard retrieves all threads, complete with the first 5 posts
func GetAllBoard() (common.Board, error) {
	r, err := prepared["get_all_thread_ids"].Query()
	if err != nil {
		return nil, err
	}
	return scanBoard(r)
}

// GetRecentPosts retrieves posts created in the thread in the last 15 minutes.
// Posts that are being editted also have their Body property set.
func GetRecentPosts(op uint64) (posts []PostStats, err error) {
	r, err := prepared["get_recent_posts"].Query(op)
	if err != nil {
		return
	}
	defer r.Close()

	posts = make([]PostStats, 0, 64)
	var p PostStats
	for r.Next() {
		err = r.Scan(&p.ID, &p.Time, &p.Editing, &p.HasImage)
		if err != nil {
			return
		}
		posts = append(posts, p)
	}
	err = r.Err()
	if err != nil {
		return
	}

	// Get open post bodies
	if len(posts) != 0 {
		var tx *bolt.Tx
		tx, err = boltDB.Begin(false)
		if err != nil {
			return
		}
		defer tx.Rollback()

		buc := tx.Bucket([]byte("open_bodies"))
		for i, p := range posts {
			if !p.Editing {
				continue
			}
			// Buffer is only valid for the transaction. Need to copy.
			posts[i].Body = util.CloneBytes(buc.Get(formatPostID(p.ID)))
		}
	}

	return
}

func scanCatalog(table tableScanner) (board common.Board, err error) {
	defer table.Close()
	board = make(common.Board, 0, 8)

	for table.Next() {
		var (
			t    common.Thread
			post postScanner
			img  imageScanner
		)

		args := make([]interface{}, 0, 30)
		args = append(args,
			&t.Board, &t.PostCtr, &t.ImageCtr, &t.ReplyTime, &t.BumpTime,
			&t.Subject,
		)
		args = append(args, post.ScanArgs()...)
		args = append(args, img.ScanArgs()...)
		err = table.Scan(args...)
		if err != nil {
			return
		}

		t.Post, err = post.Val()
		if err != nil {
			return
		}
		t.Image = img.Val()
		if t.Image != nil {
			t.Image.Spoiler, t.Image.Name = post.Image()
		}

		board = append(board, t)
	}
	err = table.Err()
	if err != nil {
		return
	}

	open := make([]*common.Post, 0, 16)
	for i := range board {
		if board[i].Editing {
			open = append(open, &board[i].Post)
		}
	}
	err = injectOpenBodies(open)

	return
}

func scanBoard(table tableScanner) (board common.Board, err error) {
	defer table.Close()

	ids := make([]uint64, 0, 32)
	for table.Next() {
		var id uint64
		err = table.Scan(&id)
		if err != nil {
			return
		}
		ids = append(ids, id)
	}
	err = table.Err()
	if err != nil {
		return
	}

	board = make(common.Board, 0, len(ids))
	for _, id := range ids {
		var thread common.Thread
		thread, err = GetThread(id, 5)
		switch err {
		case nil:
			board = append(board, thread)
		case sql.ErrNoRows: // Deleted thread
			err = nil
		default:
			return
		}
	}

	open := make([]*common.Post, 0, 32)
	for i := range board {
		t := &board[i]
		if t.Editing {
			open = append(open, &t.Post)
		}
		for i := range t.Posts {
			if t.Posts[i].Editing {
				open = append(open, &t.Posts[i])
			}
		}
	}

	return
}

// Inject open post bodies from the embedded database into the posts
func injectOpenBodies(posts []*common.Post) error {
	if len(posts) == 0 {
		return nil
	}

	tx, err := boltDB.Begin(false)
	if err != nil {
		return err
	}

	buc := tx.Bucket([]byte("open_bodies"))
	sort.Sort(postSorter(posts))
	for _, p := range posts {
		p.Body = string(buc.Get(formatPostID(p.ID)))
	}

	return tx.Rollback()
}
