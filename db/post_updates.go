package db

import (
	"database/sql"
	"encoding/json"
	"sort"

	"github.com/bakape/meguca/common"
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
	return InTransaction(func(tx *sql.Tx) (err error) {
		err = populateCommands(tx, board, com)
		if err != nil {
			return
		}
		_, err = sq.Update("posts").
			SetMap(map[string]interface{}{
				"editing":  false,
				"body":     body,
				"commands": commandRow(com),
				"password": nil,
			}).
			Where("id = ?", id).
			RunWith(tx).
			Exec()
		if err != nil {
			return
		}
		err = writeLinks(tx, id, links)
		return
	})
}

// Write post links to database. Invalid links are simply not written
func writeLinks(tx *sql.Tx, source uint64, links []uint64) (err error) {
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

	q, err := tx.Prepare(
		`insert into links (source, target)
			select $1, $2
			where exists (select from posts p where p.id = $2)`,
	)
	if err != nil {
		return
	}

	for _, id := range links {
		_, err = q.Exec(source, id)
		if err != nil {
			return
		}
	}
	return
}
