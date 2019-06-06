package db

import (
	"database/sql"
	"fmt"
	"strconv"

	"github.com/Masterminds/squirrel"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/lib/pq"
)

const (
	postSelectsSQL = `p.editing, p.moderated, p.spoiler, p.sage, p.id, p.page,
	p.time, p.body, p.flag, p.name, p.trip, p.auth,
	(select array_agg((l.target, linked_post.op, linked_thread.board))
		from links as l
		join posts as linked_post on l.target = linked_post.id
		join threads as linked_thread on linked_post.op = linked_thread.id
		where l.source = p.id
	),
	p.commands, p.imageName,
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
	t.update_time, t.bump_time, t.subject, t.locked, ` + postSelectsSQL

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
// sql.Scanner.Scan()
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
		&p.Editing, &p.Moderated, &p.spoiler, &p.Sage, &p.ID, &p.Page,
		&p.Time, &p.Body, &p.Flag, &p.Name, &p.Trip, &p.Auth, &p.links,
		&p.commands, &p.imageName,
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
			cap = int(t.PostCount)
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

		// Inject  moderation into affected posts
		moderated := make([]*common.Post, 0, 64)
		filterModerated(&moderated, &t.Post)
		for i := range t.Posts {
			filterModerated(&moderated, &t.Posts[i])
		}
		return injectModeration(moderated, tx)
	})
	if err != nil {
		return
	}

	// Inject bodies into open posts
	open := make([]*common.Post, 0, 64)
	filterOpen(&open, &t.Post)
	for i := range t.Posts {
		filterOpen(&open, &t.Posts[i])
	}
	err = injectOpenBodies(open)
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
		&t.Sticky, &t.Board, &t.PostCount, &t.ImageCount, &t.UpdateTime,
		&t.BumpTime, &t.Subject, &t.Locked,
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
		err = injectModeration([]*common.Post{&res.Post}, nil)
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
	b, err = scanCatalog(getOPs().
		Where("t.board = ?", board).
		OrderBy("sticky desc, bump_time desc"))
	return
}

// GetThreadIDs retrieves all threads IDs on the board in bump order with stickies first
func GetThreadIDs(board string) ([]uint64, error) {
	return scanThreadIDs(sq.Select("id").
		From("threads").
		Where("board = ?", board).
		OrderBy("sticky desc, bump_time desc"))
}

// GetAllBoardCatalog retrieves all threads for the "/all/" meta-board
func GetAllBoardCatalog() (board common.Board, err error) {
	board, err = scanCatalog(getOPs().OrderBy("bump_time desc"))
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
	return scanThreadIDs(sq.Select("id").
		From("threads").
		OrderBy("bump_time desc"))
}

func scanCatalog(q squirrel.SelectBuilder) (board common.Board, err error) {
	board.Threads = make([]common.Thread, 0, 32)
	err = queryAll(q, func(r *sql.Rows) (err error) {
		t, err := scanOP(r)
		if err != nil {
			return
		}
		board.Threads = append(board.Threads, t)
		return
	})
	if err != nil {
		return
	}

	open := make([]*common.Post, 0, 16)
	moderated := make([]*common.Post, 0, 16)
	for i := range board.Threads {
		ptr := &board.Threads[i].Post
		filterOpen(&open, ptr)
		filterModerated(&moderated, ptr)
	}
	err = injectOpenBodies(open)
	if err != nil {
		return
	}
	err = injectModeration(moderated, nil)
	return
}

func scanThreadIDs(q squirrel.SelectBuilder) (ids []uint64, err error) {
	ids = make([]uint64, 0, 64)
	err = queryAll(q, func(r *sql.Rows) (err error) {
		var id uint64
		err = r.Scan(&id)
		if err != nil {
			return
		}
		ids = append(ids, id)
		return
	})
	return
}

// Filter and append a post if it has injectable open bodies
func filterOpen(open *[]*common.Post, p *common.Post) {
	if p.Editing {
		*open = append(*open, p)
	}
}

// Filter and append a post if it has injectable moderation
func filterModerated(moderated *[]*common.Post, p *common.Post) {
	if p.Moderated {
		*moderated = append(*moderated, p)
	}
}

// Inject moderation information into affected post structs.
// tx is optional.
func injectModeration(posts []*common.Post, tx *sql.Tx) (err error) {
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

	q := sq.Select("post_id", "type", "length", "by", "data").
		From("post_moderation").
		Where(fmt.Sprintf("post_id in %s", string(set)))
	if tx != nil {
		q = q.RunWith(tx)
	}
	r, err := q.Query()
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
