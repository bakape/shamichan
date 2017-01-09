package db

import "github.com/bakape/meguca/common"
import "database/sql"
import "github.com/lib/pq"
import "encoding/json"
import "bytes"
import "strconv"

type tableScanner interface {
	rowScanner
	Next() bool
}

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
	name, trip, auth sql.NullString
	commands         pq.StringArray
}

func (p *postScanner) ScanArgs() []interface{} {
	return []interface{}{
		&p.Editing, &p.ID, &p.Time, &p.Body, &p.name, &p.trip, &p.auth,
		&p.commands,
	}
}

func (p *postScanner) Val() (common.Post, error) {
	p.Name = p.name.String
	p.Trip = p.trip.String
	p.Auth = p.auth.String

	if p.commands != nil {
		p.Commands = make([]common.Command, len(p.commands))
		for i := range p.commands {
			err := json.Unmarshal([]byte(p.commands[i]), &p.Commands[i])
			if err != nil {
				return p.Post, err
			}
		}
	}

	return p.Post, nil
}

type linksRow struct {
	source, target, targetOP uint64
	targetBoard              string
}

// // GetThread retrieves public thread data from the database
// func GetThread(id uint64, lastN int) (common.Thread, error) {
// 	q := r.
// 		Table("threads").
// 		GetAll(id). // Can not join after Get(). Meh.
// 		EqJoin("id", r.Table("posts")).
// 		Zip()

// 	getPosts := r.
// 		Table("posts").
// 		GetAllByIndex("op", id).
// 		Filter(filterDeleted).
// 		OrderBy("id").
// 		CoerceTo("array")

// 	// Only fetch last N number of replies
// 	if lastN != 0 {
// 		getPosts = getPosts.Slice(-lastN)
// 	}

// 	q = q.Merge(map[string]r.Term{
// 		"posts":       getPosts.Without(omitForThreadPosts),
// 		"lastUpdated": getLastUpdated,
// 	}).
// 		Without("ip", "op", "password")

// 	var thread common.Thread
// 	if err := One(q, &thread); err != nil {
// 		return thread, err
// 	}

// 	if thread.Deleted {
// 		return common.Thread{}, r.ErrEmptyResult
// 	}

// 	// Remove OP from posts slice to prevent possible duplication. Post might
// 	// be deleted before the thread due to a deletion race.
// 	if len(thread.Posts) != 0 && thread.Posts[0].ID == id {
// 		thread.Posts = thread.Posts[1:]
// 	}

// 	thread.Abbrev = lastN != 0

// 	return thread, nil
// }

// GetPost reads a single post from the database
func GetPost(id uint64) (res common.StandalonePost, err error) {
	var (
		args = make([]interface{}, 2, 20)
		post postScanner
		img  imageScanner
	)
	args[0] = &res.OP
	args[1] = &res.Board
	args = append(args, post.ScanArgs()...)
	args = append(args, img.ScanArgs()...)

	err = prepared["getPost"].QueryRow(id).Scan(args...)
	if err != nil {
		return
	}
	res.Post, err = post.Val()
	if err != nil {
		return
	}
	res.Image = img.Val()

	links, backlinks, err := getLinks(id)
	if err != nil {
		return
	}
	res.Links = repackLinks(links)
	res.Backlinks = repackLinks(backlinks)

	return res, nil
}

func repackLinks(rows []linksRow) common.LinkMap {
	if rows == nil {
		return nil
	}
	links := make(common.LinkMap, len(rows))
	for _, r := range rows {
		links[r.target] = common.Link{
			OP:    r.targetOP,
			Board: r.targetBoard,
		}
	}
	return links
}

// GetBoard retrieves all OPs of a single board
func GetBoard(board string) (common.Board, error) {
	r, err := prepared["getBoard"].Query(board)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return scanBoard(r)
}

// GetAllBoard retrieves all threads for the "/all/" meta-board
func GetAllBoard() (common.Board, error) {
	r, err := prepared["getAllBoard"].Query()
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
		)

		args := make([]interface{}, 0, 23)
		args = append(args,
			&t.Board, &t.ID, &t.PostCtr, &t.ImageCtr, &t.ReplyTime, &t.Subject,
			&img.Spoiler, &t.Time, &name, &trip, &auth, &img.Name, &t.LogCtr,
		)
		args = append(args, img.ScanArgs()...)
		err := table.Scan(args...)
		if err != nil {
			return nil, err
		}

		t.Name = name.String
		t.Trip = trip.String
		t.Auth = auth.String
		t.Image = img.Val()

		// Allocate more space in advance, to reduce backing array reallocation
		if len(board) == cap(board) {
			new := make(common.Board, len(board), cap(board)*2)
			copy(new, board)
			board = new
		}

		board = append(board, t)
	}

	return board, nil
}

// Retrieve links and backlinks by post IDs
func getLinks(ids ...uint64) (
	links, backlinks []linksRow, err error,
) {
	// Stringify ID array
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, id := range ids {
		if i != 0 {
			buf.WriteByte(',')
		}
		buf.WriteString(strconv.FormatUint(id, 10))
	}
	buf.WriteByte('}')
	args := buf.String()

	links, err = scanLinks("getLinks", args)
	if err != nil {
		return
	}
	backlinks, err = scanLinks("getBacklinks", args)
	return
}

func scanLinks(queryID, ids string) ([]linksRow, error) {
	r, err := prepared[queryID].Query(ids)
	if err != nil {
		return nil, err
	}

	links := make([]linksRow, 0, 8)
	for r.Next() {
		var row linksRow
		err := r.Scan(&row.targetBoard, &row.source, &row.target, &row.targetOP)
		if err != nil {
			return nil, err
		}

		if len(links) == cap(links) {
			new := make([]linksRow, len(links), cap(links)*2)
			copy(new, links)
			links = new
		}

		links = append(links, row)
	}

	if len(links) == 0 {
		return nil, nil
	}
	return links, nil
}
