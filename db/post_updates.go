package db

import (
	"encoding/json"
	"sort"

	"github.com/bakape/meguca/common"
	"github.com/jackc/pgx"
)

// Data populated on post closure
type CloseData struct {
	Links    map[uint64]common.Link
	Commands json.RawMessage
}

// ClosePost closes an open post and validates and commits any links and
// hash commands
func ClosePost(
	id uint64,
	board, body string,
	links []uint64,
	com []common.Command,
) (err error) {
	return InTransaction(func(tx *pgx.Tx) (err error) {
		err = populateCommands(tx, board, com)
		if err != nil {
			return
		}
		_, err = tx.Exec(
			`close_post`,
			id,
			body,
			commandRow(com),
		)
		if err != nil {
			return
		}
		err = writeLinks(tx, id, links)
		return
	})
}

// Write post links to database. Invalid links are simply not written
func writeLinks(tx *pgx.Tx, source uint64, links []uint64) (err error) {
	if len(links) == 0 {
		return
	}

	// Dedup to avoid extra DB queries
	exist := make(map[uint64]struct{}, len(links))
	for _, id := range links {
		exist[id] = struct{}{}
	}
	if len(links) != len(exist) {
		// Reuse same slice
		links = links[:0]
		for id := range exist {
			links = append(links, id)
		}
	}

	// Sort for less page missed on the DB side
	sort.Sort(idSorter(links))

	for _, id := range links {
		_, err = tx.Exec("insert_link", source, id)
		if err != nil {
			return
		}
	}
	return
}
