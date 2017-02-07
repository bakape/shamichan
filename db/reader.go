package db

import (
	"database/sql"

	"github.com/bakape/meguca/common"
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
	banned, spoiler             sql.NullBool
	name, trip, auth, imageName sql.NullString
	links, backlinks            linkRow
	commands                    commandRow
}

func (p *postScanner) ScanArgs() []interface{} {
	return []interface{}{
		&p.Editing, &p.banned, &p.spoiler, &p.ID, &p.Time, &p.Body, &p.name, &p.trip,
		&p.auth, &p.links, &p.backlinks, &p.commands, &p.imageName,
	}
}

func (p postScanner) Val() (common.Post, error) {
	p.Banned = p.banned.Bool
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
	var logCtr sql.NullInt64
	err = row.Scan(
		&t.Board, &t.PostCtr, &t.ImageCtr, &t.ReplyTime, &t.BumpTime,
		&t.Subject, &logCtr,
	)
	if err != nil {
		return
	}
	t.Abbrev = lastN != 0
	t.LogCtr = uint64(logCtr.Int64)

	// Get OP post. Need to fetch separately, in case, if not fetching the full
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

	return
}

func scanThreadPost(rs rowScanner) (res common.Post, err error) {
	var (
		args = make([]interface{}, 0, 21)
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
		args = make([]interface{}, 2, 25)
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

	return res, nil
}

// GetBoard retrieves all OPs of a single board
func GetBoard(board string) (common.Board, error) {
	r, err := prepared["get_board"].Query(board)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return scanBoard(r)
}

// GetAllBoard retrieves all threads for the "/all/" meta-board
func GetAllBoard() (common.Board, error) {
	r, err := prepared["get_all_board"].Query()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return scanBoard(r)
}

func scanBoard(table tableScanner) (common.Board, error) {
	board := make(common.Board, 0, 8)

	for table.Next() {
		var (
			t                common.BoardThread
			name, trip, auth sql.NullString
			img              imageScanner
			logCtr           sql.NullInt64
		)

		args := make([]interface{}, 0, 24)
		args = append(args,
			&t.Board, &t.ID, &t.PostCtr, &t.ImageCtr, &t.ReplyTime, &t.BumpTime,
			&t.Subject, &img.Spoiler, &t.Time, &name, &trip, &auth, &img.Name,
			&logCtr,
		)
		args = append(args, img.ScanArgs()...)
		err := table.Scan(args...)
		if err != nil {
			return nil, err
		}

		t.Name = name.String
		t.Trip = trip.String
		t.Auth = auth.String
		t.LogCtr = uint64(logCtr.Int64)
		t.Image = img.Val()

		board = append(board, t)
	}

	return board, table.Err()
}
