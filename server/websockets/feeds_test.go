package websockets

import (
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

func (*DB) TestStreamUpdates(c *C) {
	thread := types.DatabaseThread{
		ID:  1,
		Log: [][]byte{},
	}
	c.Assert(db.Write(r.Table("threads").Insert(thread)), IsNil)

	// Empty log
	init, read, cursor, err := streamUpdates(1, 0)
	c.Assert(err, IsNil)
	c.Assert(init, DeepEquals, [][]byte{})

	// Update
	log := [][]byte{[]byte("foo"), []byte("bar")}
	q := r.Table("threads").Get(1).Update(map[string][][]byte{
		"log": [][]byte{log[0]},
	})
	c.Assert(db.Write(q), IsNil)

	q = r.Table("threads").Get(1).Update(map[string]r.Term{
		"log": appendLog(log[1]),
	})
	c.Assert(db.Write(q), IsNil)

	c.Assert(<-read, DeepEquals, log[0])
	c.Assert(<-read, DeepEquals, log[1])
	c.Assert(cursor.Close(), IsNil)

	// Existing data
	init, _, cursor, err = streamUpdates(1, 1)
	c.Assert(err, IsNil)
	c.Assert(init, DeepEquals, [][]byte{log[1]})
	c.Assert(cursor.Close(), IsNil)
}
