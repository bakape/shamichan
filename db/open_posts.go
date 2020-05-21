package db

import (
	"context"
	"encoding/json"
	"sort"

	"github.com/jackc/pgx/v4"
)

// Write open post bodies to DB.
// Bodies must be in map[string]Body JSON map format.
func WriteOpenPostBodies(buf []byte) (err error) {
	var bodies map[uint64]json.RawMessage
	err = json.Unmarshal(buf, &bodies)
	if err != nil {
		return
	}

	// Sort IDs for more sequential DB access
	toWrite := make(idSorter, 0, len(bodies))
	for id := range bodies {
		toWrite = append(toWrite, id)
	}
	sort.Sort(toWrite)

	return InTransaction(context.Background(), func(tx pgx.Tx) (err error) {
		for _, id := range toWrite {
			_, err = tx.Exec(
				context.Background(),
				`update posts
				set body = $1
				where id = $2 and editing = true`,
				bodies[id],
				id,
			)
			if err != nil {
				return
			}
		}
		return
	})
}
