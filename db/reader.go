package db

import (
	"context"

	"github.com/jackc/pgx"
)

var (
	// Don't reallocate this
	emptyArray = []byte("[]")
)

// The PL/pgSQL functions return null on non-existence. Cast that to
// pgx.ErrNoRows.
func castNoRows(buf *[]byte, err *error) {
	if *err == nil && len(*buf) == 0 {
		*err = pgx.ErrNoRows
	}
}

// GetPost reads a single post from the database
func GetPost(ctx context.Context, id uint64) (post []byte, err error) {
	err = db.
		QueryRow(
			ctx,
			`select encode(p)
			from posts p
			where p.id = $1`,
			id,
		).
		Scan(&post)
	castNoRows(&post, &err)
	return
}

// // GetThread retrieves public thread data from the database.
// // page: page of the thread to fetch. -1 to fetch the last page.
// func GetThread(id uint64, page int) (thread []byte, err error) {
// 	err = db.QueryRow("select get_thread($1, $2)", id, page).Scan(&thread)
// 	castNoRows(&thread, &err)
// 	return
// }

// // Ensure buf is always an array
// func ensureArray(buf *[]byte) {
// 	if len(*buf) == 0 {
// 		*buf = emptyArray
// 	}
// }

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
