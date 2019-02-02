package db

import (
	"database/sql"
	"fmt"
	"strconv"

	"meguca/common"
	"meguca/config"

	"github.com/Masterminds/squirrel"
	"github.com/lib/pq"
)

const (
	postSelectsSQL = `p.editing, p.moderated, p.spoiler, p.sage, p.id,
	p.time, p.body, p.flag, p.name, p.trip, p.auth,
	(select array_agg((l.target, linked_post.op, linked_thread.board))
		from links as l
		join posts as linked_post on l.target = linked_post.id
		join threads as linked_thread on linked_post.op = linked_thread.id
		where l.source = p.id
	),
	p.commands, p.imageName,
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
	t.replyTime, t.bumpTime, t.subject, t.locked, ` + postSelectsSQL

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
	Audio, Video, Spoiler             sql.NullBool
	FileType, ThumbType, Length, Size sql.NullInt64
	Name, SHA1, MD5, Title, Artist    sql.NullString
	Dims                              pq.Int64Array
}

// Returns and array of pointers to the struct fields for passing to
// rowScanner.Scan()
func (i *imageScanner) ScanArgs() []interface{} {
	return []interface{}{
		&i.Audio, &i.Video, &i.FileType, &i.ThumbType, &i.Dims,
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
	spoiler   bool
	imageName string
	links     linkScanner
	commands  commandRow
}

func (p *postScanner) ScanArgs() []interface{} {
	return []interface{}{
		&p.Editing, &p.Moderated, &p.spoiler, &p.Sage, &p.ID, &p.Time, &p.Body,
		&p.Flag, &p.Name, &p.Trip, &p.Auth, &p.links, &p.commands,
		&p.imageName, &p.PosterID,
	}
}

func (p postScanner) Val() (common.Post, error) {
	p.Links = []common.Link(p.links)
	p.Commands = []common.Command(p.commands)

	return p.Post, nil
}

// Returns if image is spoiled and it's assigned name, if any
func (p postScanner) Image() (bool, string) {
	return p.spoiler, p.imageName
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
	err = InTransaction(true, func(tx *sql.Tx) (err error) {
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
		return r.Err()
	})
	if err != nil {
		return
	}

	// Inject bodies and moderation into open posts
	open := make([]*common.Post, 0, 64)
	moderated := make([]*common.Post, 0, 64)
	filterInjectable(&open, &moderated, &t.Post)
	for i := range t.Posts {
		filterInjectable(&open, &moderated, &t.Posts[i])
	}
	err = injectOpenBodies(open)
	if err != nil {
		return
	}
	err = injectModeration(moderated)
	return
}

func scanOP(r rowScanner) (t common.Thread, err error) {
	var (
		post  postScanner
		img   imageScanner
		pArgs = post.ScanArgs()
		iArgs = img.ScanArgs()
		args  = make([]interface{}, 0, 8+len(pArgs)+len(iArgs))
	)
	args = append(args,
		&t.Sticky, &t.Board, &t.PostCtr, &t.ImageCtr, &t.ReplyTime, &t.BumpTime,
		&t.Subject, &t.Locked,
	)
	args = append(args, pArgs...)
	args = append(args, iArgs...)

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
		post  postScanner
		img   imageScanner
		pArgs = post.ScanArgs()
		iArgs = img.ScanArgs()
		args  = make([]interface{}, 2, 2+len(pArgs)+len(iArgs))
	)
	args[0] = &res.OP
	args[1] = &res.Board
	args = append(args, pArgs...)
	args = append(args, iArgs...)

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

	if res.Editing {
		res.Body, err = GetOpenBody(res.ID)
		if err != nil {
			return
		}
	}
	if res.Moderated {
		err = injectModeration([]*common.Post{&res.Post})
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
	return
}

// GetThreadIDs retrieves all threads IDs on the board in bump order with stickies first
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

	return
}

// GetAllThreadsIDs retrieves all threads IDs in bump order
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
	moderated := make([]*common.Post, 0, 16)
	for i := range board.Threads {
		filterInjectable(&open, &moderated, &board.Threads[i].Post)
	}
	err = injectOpenBodies(open)
	if err != nil {
		return
	}
	err = injectModeration(moderated)
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

// Filter and append a post if it has injectable open bodies and/or moderation
func filterInjectable(open, moderated *[]*common.Post, p *common.Post) {
	if p.Editing {
		*open = append(*open, p)
	}
	if p.Moderated {
		*moderated = append(*moderated, p)
	}
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

// Inject moderation information into affected post structs
func injectModeration(posts []*common.Post) (err error) {
	if len(posts) == 0 {
		return
	}

	byID := make(map[uint64]*common.Post, len(posts))
	set := make([]byte, 1, 512)
	set[0] = '('
	for i, p := range posts {
		byID[p.ID] = p
		if i != 0 {
			set = append(set, ',')
		}
		set = strconv.AppendUint(set, p.ID, 10)
	}
	set = append(set, ')')

	r, err := sq.Select("post_id", "type", "length", "by", "data").
		From("post_moderation").
		Where(fmt.Sprintf("post_id in %s", string(set))).
		Query()
	if err != nil {
		return
	}
	defer r.Close()

	var (
		e  common.ModerationEntry
		id uint64
	)
	for r.Next() {
		err = r.Scan(&id, &e.Type, &e.Length, &e.By, &e.Data)
		if err != nil {
			return
		}
		byID[id].Moderation = append(byID[id].Moderation, e)
	}

	return r.Err()
}
