package db

import (
	"fmt"
	"strings"

	"github.com/bakape/meguca/config"
)

// GetThread retrieves public thread data from the database.
// page: page of the thread to fetch.
// 	-1 to fetch the last page.
// 	-5 to fetch last 5 posts
func GetThread(id uint64, page int) (thread []byte, err error) {
	err = db.QueryRow("select get_thread($1, $2)", id, page).Scan(&thread)
	return
}

// GetPost reads a single post from the database
func GetPost(id uint64) (post []byte, err error) {
	err = sq.
		Select(
			`encode_post(p)
			|| jsonb_build_object(
				'op', p.op,
				'board', post_board(p.id)
			)`,
		).
		From("posts p").
		Where("p.id = ?", id).
		QueryRow().
		Scan(&post)
	return
}

// GetBoardCatalog retrieves all OPs of a single board
func GetBoardCatalog(board string) (buf []byte, err error) {
	err = sq.
		Select(
			`jsonb_agg(
				get_thread(id, -6) - 'page'
				order by sticky desc, bump_time desc
			)`,
		).
		From("threads").
		Where("board = ?", board).
		QueryRow().
		Scan(&buf)
	return
}

// GetAllBoardCatalog retrieves all threads for the "/all/" meta-board
func GetAllBoardCatalog() (buf []byte, err error) {
	q := sq.
		Select(
			`jsonb_agg(
			get_thread(id, -6) - 'page'
			order by bump_time desc
		)`,
		).
		From("threads")

	// Hide threads from NSFW boards, if enabled
	if config.Get().HideNSFW {
		// TODO:  Test this

		var w strings.Builder
		first := true
		for _, b := range config.GetAllBoardConfigs() {
			if !b.NSFW {
				continue
			}
			if first {
				w.WriteByte('(')
				first = false
			} else {
				w.WriteByte(',')
			}
			fmt.Fprintf(&w, `'%s'`, b.ID)
		}
		if !first {
			// Don't allocate for empty filter set
			w.WriteByte(')')
		}

		if !first {
			// Something actually written
			q = q.Where("board not in " + w.String())
		}
	}

	err = q.QueryRow().Scan(&buf)
	return
}

// Retrieves all threads for a specific board on a specific page
func GetBoard(board string, page uint) (data []byte, err error) {
	err = db.QueryRow(`select get_board($1, $2)`, board, page).Scan(&data)
	return
}

// Retrieves all threads for the "/all/" meta-board on a specific page
func GetAllBoard(page uint) (board []byte, err error) {
	err = db.QueryRow(`select get_all_board($1)`, page).Scan(&board)
	return
}
