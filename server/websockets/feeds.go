package websockets

import (
	"github.com/bakape/meguca/db"
	r "github.com/dancannon/gorethink"
)

// StreamUpdates produces a stream of the replication log updates for the
// specified thread and sends it on read. Close the cursor to stop receiving
// updates. The intial truncated contents of the log are returned emediately.
func streamUpdates(id int64, ctr uint64) (
	initial [][]byte, read <-chan []byte, cursor *r.Cursor, err error,
) {
	cursor, err = r.
		Table("threads").
		Get(id).
		Changes(r.ChangesOpts{IncludeInitial: true}).
		Map(r.Branch(
			r.Row.HasFields("old_val"),
			r.Row.
				Field("new_val").
				Field("log").
				AtIndex(r.Row.Field("old_val").Field("log").Count()).
				Default(nil),
			r.Row.Field("new_val").Field("log").Slice(ctr),
		)).
		Run(db.RSession)
	if err != nil {
		return
	}

	if !cursor.Next(&initial) {
		err = cursor.Err()
		if err != nil {
			return
		}
	}

	re := make(chan []byte) // Avoid automatic type cast to receive-only
	read = re
	cursor.Listen(re)

	return
}
