package db

import (
	"encoding/json"

	"github.com/bakape/meguca/common"
	"github.com/jackc/pgx"
)

var (
	// Don't reallocate this
	emptyArray = []byte("[]")
)

// Open post meta information
type OpenPostMeta struct {
	HasImage  bool   `json:"has_image,omitempty"`
	Spoilered bool   `json:"spoilered,omitempty"`
	Page      uint32 `json:"page"`
	Body      string `json:"body"`
}

// // Populate OpenPostMeta from post data
// func OpenPostMetaFromPost(p common.Post) (m OpenPostMeta) {
// 	m = OpenPostMeta{
// 		Page: p.Page,
// 		Body: p.Body,
// 	}
// 	if p.Image != nil {
// 		m.HasImage = true
// 		m.Spoilered = p.Image.Spoiler
// 	}
// 	return
// }

// GetThread retrieves public thread data from the database.
// page: page of the thread to fetch. -1 to fetch the last page.
func GetThread(id uint64, page int) (thread []byte, err error) {
	err = db.QueryRow("select get_thread($1, $2)", id, page).Scan(&thread)
	castNoRows(&thread, &err)
	return
}

// The PL/pgSQL functions return null on non-existence. Cast that to
// pgx.ErrNoRows.
func castNoRows(buf *[]byte, err *error) {
	if *err == nil && len(*buf) == 0 {
		*err = pgx.ErrNoRows
	}
}

// GetPost reads a single post from the database
func GetPost(id uint64) (post []byte, err error) {
	err = db.
		QueryRow(
			`select encode_post(p)
				|| jsonb_build_object(
					'op', p.op,
					'board', post_board(p.id)
				)
			from posts p
			where p.id = $1`,
			id,
		).
		Scan(&post)
	castNoRows(&post, &err)
	return
}

// TODO: Get all board index
// TODO: Get all board catalog

// Ensure buf is always an array
func ensureArray(buf *[]byte) {
	if len(*buf) == 0 {
		*buf = emptyArray
	}
}

// // GetAllBoardCatalog retrieves all threads for the "/all/" meta-board
// func GetAllBoardCatalog() (buf []byte, err error) {
// 	err = db.
// 		QueryRow(
// 			`select jsonb_agg(
// 				get_thread(id, -6) - 'page'
// 				order by bump_time desc
// 			)
// 			from threads`,
// 		).
// 		Scan(&buf)
// 	ensureArray(&buf)
// 	return
// }

// Get thread meta-information for initializing thread update feeds
func GetThreadMeta(thread uint64) (
	all map[uint64]uint32,
	open map[uint64]OpenPostMeta,
	moderation map[uint64][]common.ModerationEntry,
	err error,
) {
	// Ensure any pending post body changes for this thread (and also others,
	// while we are at it) are flushed to DB before read
	err = FlushOpenPostBodies()
	if err != nil {
		return
	}

	// TODO: Move this to SQL or PL/pgSQL
	// var buf [3][]byte
	// err = db.
	// 	QueryRow(
	// 		`select
	// 		(),
	// 		(),
	// 		()`,
	// 	).
	// 	Scan(&buf[0], &buf[1], &buf[2])
	// return

	all = make(map[uint64]uint32, 1<<10)
	open = make(map[uint64]OpenPostMeta)
	moderation = make(map[uint64][]common.ModerationEntry)

	err = InTransaction(func(tx *pgx.Tx) (err error) {
		var r *pgx.Rows
		defer func() {
			if r != nil {
				r.Close()
			}
		}()

		r, err = tx.Query(
			`select id page
			from posts
			where op = $1`,
			thread,
		)
		if err != nil {
			return
		}

		var (
			id   uint64
			page uint32
		)
		for r.Next() {
			err = r.Scan(&id, &page)
			if err != nil {
				return
			}
			all[id] = page
		}
		err = r.Err()
		if err != nil {
			return
		}
		r.Close()

		r, err = tx.Query(
			`select id, sha1 is not null, spoiler, page
			from posts
			where op = $1 and editing = true`,
			thread,
		)
		if err != nil {
			return
		}

		var p OpenPostMeta
		for r.Next() {
			err = r.Scan(&id, &p.HasImage, &p.Spoilered, &p.Page)
			if err != nil {
				return
			}
			open[id] = p
		}
		err = r.Err()
		if err != nil {
			return
		}
		r.Close()

		r, err = tx.Query(
			`select id, get_post_moderation(id)
			from posts
			where op = $1 abd moderated = true`,
			thread,
		)
		if err != nil {
			return
		}

		var (
			buf []byte
			mod []common.ModerationEntry
		)
		for r.Next() {
			err = r.Scan(&id, &buf)
			if err != nil {
				return
			}
			err = json.Unmarshal(buf, &mod)
			if err != nil {
				return
			}
			copy(moderation[id], mod)
		}
		return r.Err()
	})
	return
}

// TODO: Board meta for board update feeds.

// Get data assigned on post closure like links and hash command results
func GetPostCloseData(id uint64) (data CloseData, err error) {
	err = db.
		QueryRow(
			`select jsonb_build_object(
				'links', get_links(id),
				'commands', commands
			)
			from posts
			where id = $1`,
		).
		Scan(&data)
	return
}
