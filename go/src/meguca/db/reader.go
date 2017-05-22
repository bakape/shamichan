package db

import (
	"database/sql"
	"meguca/common"
	"meguca/util"

	"github.com/boltdb/bolt"
	"github.com/lib/pq"
)

type imageScanner struct {
	APNG, Audio, Video, Spoiler       sql.NullBool
	FileType, ThumbType, Length, Size sql.NullInt64
	Name, SHA1, MD5, Title, Artist    sql.NullString
	Dims                              pq.Int64Array
}

// Returns and array of pointers to the struct fields for passing to
// rowScanner.Scan()
func (i *imageScanner) ScanArgs() []interface{} {
	return []interface{}{
		&i.APNG, &i.Audio, &i.Video, &i.FileType, &i.ThumbType, &i.Dims,
		&i.Length, &i.Size, &i.MD5, &i.SHA1, &i.Title, &i.Artist,
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
			Title:     i.Title.String,
			Artist:    i.Artist.String,
		},
		Name: i.Name.String,
	}
}

type postScanner struct {
	common.Post
	banned, spoiler, deleted, sage sql.NullBool
	name, trip, auth, imageName    sql.NullString
	links, backlinks               linkRow
	commands                       commandRow
}

func (p *postScanner) ScanArgs() []interface{} {
	return []interface{}{
		&p.Editing, &p.banned, &p.spoiler, &p.deleted, &p.sage, &p.ID, &p.Time,
		&p.Body, &p.name, &p.trip, &p.auth, &p.links, &p.backlinks, &p.commands,
		&p.imageName,
	}
}

func (p postScanner) Val() (common.Post, error) {
	p.Banned = p.banned.Bool
	p.Deleted = p.deleted.Bool
	p.Sage = p.sage.Bool
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

	// Get thread metadata and OP
	t, err = scanOP(tx.Stmt(prepared["get_thread"]).QueryRow(id))
	if err != nil {
		return
	}
	t.Abbrev = lastN != 0

	// Get replies
	var (
		cap   int
		limit *int
	)
	if lastN != 0 {
		cap = lastN
		limit = &lastN
	} else {
		cap = int(t.PostCtr)
	}
	r, err := tx.Stmt(prepared["get_thread_posts"]).Query(id, limit)
	if err != nil {
		return
	}
	defer r.Close()

	// Scan replies into []common.Post
	var (
		post postScanner
		img  imageScanner
		p    common.Post
		args = append(post.ScanArgs(), img.ScanArgs()...)
	)
	t.Posts = make([]common.Post, 0, cap)
	for r.Next() {
		err = r.Scan(args...)
		if err != nil {
			return
		}
		p, err = extractPost(post, img)
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
	open := make([]*common.Post, 0, 32)
	if t.Editing {
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

func scanOP(r rowScanner) (t common.Thread, err error) {
	var (
		post postScanner
		img  imageScanner
	)

	args := make([]interface{}, 0, 34)
	args = append(args, threadScanArgs(&t)...)
	args = append(args, post.ScanArgs()...)
	args = append(args, img.ScanArgs()...)
	err = r.Scan(args...)
	if err != nil {
		return
	}

	t.Post, err = extractPost(post, img)
	return
}

func extractPost(ps postScanner, is imageScanner) (p common.Post, err error) {
	p, err = ps.Val()
	if err != nil {
		return
	}
	p.Image = is.Val()
	if p.Image != nil {
		p.Image.Spoiler, p.Image.Name = ps.Image()
	}
	return
}

// GetPost reads a single post from the database
func GetPost(id uint64) (res common.StandalonePost, err error) {
	var (
		args = make([]interface{}, 2, 29)
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
func GetBoardCatalog(board string) (b common.Board, err error) {
	r, err := prepared["get_board"].Query(board)
	if err != nil {
		return
	}
	b, err = scanCatalog(r)
	if err != nil {
		return
	}
	return
}

// Retrieves all threads IDs on the board in bump order with stickies first
func GetThreadIDs(board string) ([]uint64, error) {
	r, err := prepared["get_board_thread_ids"].Query(board)
	if err != nil {
		return nil, err
	}
	return scanThreadIDs(r)
}

// GetAllBoardCatalog retrieves all threads for the "/all/" meta-board
func GetAllBoardCatalog() (common.Board, error) {
	r, err := prepared["get_all_board"].Query()
	if err != nil {
		return nil, err
	}
	return scanCatalog(r)
}

// Retrieves all threads IDs in bump order
func GetAllThreadsIDs() ([]uint64, error) {
	r, err := prepared["get_all_thread_ids"].Query()
	if err != nil {
		return nil, err
	}
	return scanThreadIDs(r)
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
			posts[i].Body = util.CloneBytes(buc.Get(encodeUint64Heap(p.ID)))
		}
	}

	return
}

func scanCatalog(table tableScanner) (board common.Board, err error) {
	defer table.Close()
	board = make(common.Board, 0, 32)

	var t common.Thread
	for table.Next() {
		t, err = scanOP(table)
		if err != nil {
			return
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

// Return arguments for scanning a common.Thread from the DB
func threadScanArgs(t *common.Thread) []interface{} {
	return []interface{}{
		&t.Sticky, &t.Board, &t.PostCtr, &t.ImageCtr, &t.ReplyTime, &t.BumpTime,
		&t.Subject,
	}
}

func scanThreadIDs(table tableScanner) (ids []uint64, err error) {
	defer table.Close()

	ids = make([]uint64, 0, 64)
	var id uint64
	for table.Next() {
		err = table.Scan(&id)
		if err != nil {
			return
		}
		ids = append(ids, id)
	}
	err = table.Err()

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
	for _, p := range posts {
		p.Body = string(buc.Get(encodeUint64Heap(p.ID)))
	}

	return tx.Rollback()
}
