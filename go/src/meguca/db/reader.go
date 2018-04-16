package db

import (
	"database/sql"
	"meguca/common"
	"meguca/config"

	"github.com/Masterminds/squirrel"

	"github.com/lib/pq"
)

const (
	postSelectsSQL = `p.editing, p.banned, p.spoiler, p.deleted, p.sage, p.id,
	p.time, p.body, p.flag, p.name, p.trip, p.auth, p.commands, p.imageName,
	p.posterID,
	i.*`

	threadSelectsSQL = `t.sticky, t.board,
	(
		select count(*)
		from posts
		where t.id = posts.op
	),
	(
		select count(*)
		from posts
		where t.id = posts.op
			and posts.SHA1 is not null
	),
	t.replyTime, t.bumpTime, t.subject, t.nonLive, t.locked, ` + postSelectsSQL

	getOPSQL = `
	select ` + threadSelectsSQL + `
	from threads as t
	inner join posts as p on t.id = p.id
	left outer join images as i on p.SHA1 = i.SHA1
	where t.id = $1`

	getThreadPostsSQL = `
	with thread as (
		select ` + postSelectsSQL + `
		from posts as p
		left outer join images as i on p.SHA1 = i.SHA1
		where p.op = $1 and p.id != $1
		order by p.id desc
		limit $2
	)
	select * from thread
	order by id asc`
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
	banned, spoiler, deleted, sage              sql.NullBool
	name, trip, auth, imageName, flag, posterID sql.NullString
	commands                                    commandRow
}

func (p *postScanner) ScanArgs() []interface{} {
	return []interface{}{
		&p.Editing, &p.banned, &p.spoiler, &p.deleted, &p.sage, &p.ID, &p.Time,
		&p.Body, &p.flag, &p.name, &p.trip, &p.auth, &p.commands,
		&p.imageName, &p.posterID,
	}
}

func (p postScanner) Val() (common.Post, error) {
	p.Banned = p.banned.Bool
	p.Deleted = p.deleted.Bool
	p.Sage = p.sage.Bool
	p.Name = p.name.String
	p.Trip = p.trip.String
	p.Auth = p.auth.String
	p.Flag = p.flag.String
	p.PosterID = p.posterID.String
	p.Commands = []common.Command(p.commands)

	return p.Post, nil
}

// Returns if image is spoiled and it's assigned name, if any
func (p postScanner) Image() (bool, string) {
	return p.spoiler.Bool, p.imageName.String
}

// PostStats contains post open status, body and creation time
type PostStats struct {
	Editing, HasImage, Spoilered bool
	ID                           uint64
	Time                         int64
	Body                         []byte
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
	t, err = scanOP(tx.QueryRow(getOPSQL, id))
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
	r, err := tx.Query(getThreadPostsSQL, id, limit)
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

	// Fetch links
	ids := make([]uint64, len(t.Posts)+1)
	ids[0] = id
	for i := 1; i < len(ids); i++ {
		ids[i] = t.Posts[i-1].ID
	}
	links, err := getLinks(ids...)
	if err != nil {
		return
	}
	injectLinks(&t.Post, links)
	for i := 0; i < len(t.Posts); i++ {
		injectLinks(&t.Posts[i], links)
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

// Check for links belonging to post in link collection and copy any
func injectLinks(p *common.Post, links map[uint64][]common.Link) {
	if links != nil {
		p.Links = links[p.ID]
	}
}

func scanOP(r rowScanner) (t common.Thread, err error) {
	var (
		post postScanner
		img  imageScanner
	)

	args := make([]interface{}, 0, 37)
	args = append(args,
		&t.Sticky, &t.Board, &t.PostCtr, &t.ImageCtr, &t.ReplyTime, &t.BumpTime,
		&t.Subject, &t.NonLive, &t.Locked,
	)
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
		args = make([]interface{}, 2, 30)
		post postScanner
		img  imageScanner
	)
	args[0] = &res.OP
	args[1] = &res.Board
	args = append(args, post.ScanArgs()...)
	args = append(args, img.ScanArgs()...)

	err = sq.Select("p.op, p.board, "+postSelectsSQL).
		From("posts as p").
		LeftJoin("images as i on p.SHA1 = i.SHA1").
		Where("id = ?", id).
		QueryRow().
		Scan(args...)
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

	links, err := getLinks(id)
	if err != nil {
		return
	}
	injectLinks(&res.Post, links)

	if res.Editing {
		res.Body, err = GetOpenBody(res.ID)
		if err != nil {
			return
		}
	}

	return
}

func getOPs() squirrel.SelectBuilder {
	return sq.Select(threadSelectsSQL).
		From("threads as t").
		Join("posts as p on t.id = p.id").
		LeftJoin("images as i on p.SHA1 = i.SHA1")
}

// GetBoardCatalog retrieves all OPs of a single board
func GetBoardCatalog(board string) (b common.Board, err error) {
	r, err := getOPs().
		Where("t.board = ?", board).
		OrderBy("sticky desc, bumpTime desc").
		Query()
	if err != nil {
		return
	}
	b, err = scanCatalog(r)
	if err != nil {
		return
	}
	err = fetchBoardLinks(&b)
	return
}

// Retrieves all threads IDs on the board in bump order with stickies first
func GetThreadIDs(board string) ([]uint64, error) {
	r, err := sq.Select("id").
		From("threads").
		Where("board = ?", board).
		OrderBy("sticky desc, bumpTime desc").
		Query()
	if err != nil {
		return nil, err
	}
	return scanThreadIDs(r)
}

// GetAllBoardCatalog retrieves all threads for the "/all/" meta-board
func GetAllBoardCatalog() (board common.Board, err error) {
	r, err := getOPs().
		OrderBy("bumpTime desc").
		Query()
	if err != nil {
		return
	}
	board, err = scanCatalog(r)
	if err != nil {
		return
	}

	// Hide threads from NSFW boards, if enabled
	if config.Get().HideNSFW {
		filtered := make([]common.Thread, 0, len(board.Threads))
		confs := config.GetAllBoardConfigs()
		for _, t := range board.Threads {
			if !confs[t.Board].NSFW {
				filtered = append(filtered, t)
			}
		}
		board.Threads = filtered
	}

	err = fetchBoardLinks(&board)
	return
}

// Retrieves all threads IDs in bump order
func GetAllThreadsIDs() ([]uint64, error) {
	r, err := sq.Select("id").
		From("threads").
		OrderBy("bumpTime desc").
		Query()
	if err != nil {
		return nil, err
	}
	return scanThreadIDs(r)
}

func scanCatalog(table tableScanner) (board common.Board, err error) {
	defer table.Close()
	board.Threads = make([]common.Thread, 0, 32)

	var t common.Thread
	for table.Next() {
		t, err = scanOP(table)
		if err != nil {
			return
		}
		board.Threads = append(board.Threads, t)
	}
	err = table.Err()
	if err != nil {
		return
	}

	open := make([]*common.Post, 0, 16)
	for i := range board.Threads {
		if board.Threads[i].Editing {
			open = append(open, &board.Threads[i].Post)
		}
	}
	err = injectOpenBodies(open)

	return
}

// Fetch and inject board page post links
func fetchBoardLinks(b *common.Board) (err error) {
	ids := make([]uint64, 0, 512)
	for _, t := range b.Threads {
		ids = append(ids, t.ID)
		for _, p := range t.Posts {
			ids = append(ids, p.ID)
		}
	}
	links, err := getLinks(ids...)
	if err != nil {
		return
	}

	for i := range b.Threads {
		injectLinks(&b.Threads[i].Post, links)
		for j := range b.Threads[i].Posts {
			injectLinks(&b.Threads[i].Posts[j], links)
		}
	}
	return
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
