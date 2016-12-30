package db

import "github.com/bakape/meguca/common"
import "database/sql"
import "github.com/lib/pq"

type tableScanner interface {
	rowScanner
	Next() bool
}

type imageDecoder struct {
	APNG, Audio, Video, Spoiler       sql.NullBool
	FileType, ThumbType, Length, Size sql.NullInt64
	Name, SHA1, MD5                   sql.NullString
	Dims                              pq.Int64Array
}

// Returns and array of pointers to the struct fields for passing to
// rowScanner.Scan()
func (i *imageDecoder) ScanArgs() []interface{} {
	return []interface{}{
		&i.APNG, &i.Audio, &i.Video, &i.FileType, &i.ThumbType, &i.Dims,
		&i.Length, &i.Size, &i.MD5, &i.SHA1,
	}
}

// Returns the scanned *common.Image or nil, if none
func (i *imageDecoder) Val() *common.Image {
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

// // Preconstructed REQL queries that don't have to be rebuilt
// var (
// 	// Retrieves all threads for the /all/ metaboard
// 	getAllBoard = r.
// 			Table("threads").
// 			EqJoin("id", r.Table("posts")).
// 			Zip().
// 			Filter(filterDeleted).
// 			Without(omitForBoards).
// 			Merge(mergeLastUpdated).
// 			OrderBy(r.Desc("replyTime"))

// 	// Gets the most recently updated post timestamp from thread
// 	getLastUpdated = r.
// 			Table("posts").
// 			GetAllByIndex("op", r.Row.Field("id")).
// 			Field("lastUpdated").
// 			Max().
// 			Default(0)

// 	mergeLastUpdated = map[string]r.Term{
// 		"lastUpdated": getLastUpdated,
// 	}

// 	// Fields to omit in board queries. Decreases payload of DB replies.
// 	omitForBoards = []string{
// 		"body", "password", "commands", "links", "backlinks", "ip", "editing",
// 		"op",
// 	}

// 	// Fields to omit for post queries
// 	omitForPosts       = []string{"password", "ip", "lastUpdated"}
// 	omitForThreadPosts = append(omitForPosts, []string{"op", "board"}...)
// )

// func filterDeleted(p r.Term) r.Term {
// 	return p.Field("deleted").Default(false).Eq(false)
// }

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

// // GetPost reads a single post from the database
// func GetPost(id uint64) (post common.StandalonePost, err error) {
// 	q := FindPost(id).Without(omitForPosts).Default(nil)
// 	err = One(q, &post)
// 	if post.Deleted && err == nil {
// 		return common.StandalonePost{}, r.ErrEmptyResult
// 	}
// 	return
// }

// // GetBoard retrieves all OPs of a single board
// func GetBoard(board string) (data common.Board, err error) {
// 	q := r.
// 		Table("threads").
// 		GetAllByIndex("board", board).
// 		EqJoin("id", r.Table("posts")).
// 		Zip().
// 		Filter(filterDeleted).
// 		Without(omitForBoards).
// 		Merge(mergeLastUpdated).
// 		OrderBy(r.Desc("replyTime"))
// 	err = All(q, &data)

// 	// So that JSON always produces and array
// 	if data == nil {
// 		data = common.Board{}
// 	}

// 	return
// }

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
			img              imageDecoder
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
