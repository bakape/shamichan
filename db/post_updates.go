package db

import (
	"database/sql"
	"sort"

	"github.com/bakape/meguca/common"
)

// ClosePost closes an open post and validates and commits any links and
// hash commands
func ClosePost(
	id uint64,
	board, body string,
	links []uint64,
	com []common.Command,
) (err error) {
	// TODO: Propagate this with DB listener
	// TODO: Propage backlinks through update trigger
	err = InTransaction(func(tx *sql.Tx) (err error) {
		err = populateCommands(tx, board, com)
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
	if err != nil {
		return
	}

	return deleteOpenPostBody(id)
}

type idSorter []uint64

func (p idSorter) Len() int           { return len(p) }
func (p idSorter) Less(i, j int) bool { return p[i] < p[j] }
func (p idSorter) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

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
