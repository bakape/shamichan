package websockets

import (
	"encoding/json"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	r "github.com/dancannon/gorethink"
	"github.com/gorilla/websocket"
	. "gopkg.in/check.v1"
)

var _ = Suite(&DB{})

// Tests that require database access
type DB struct {
	dbName string
}

func (d *DB) SetUpSuite(c *C) {
	d.dbName = db.UniqueDBName()
	c.Assert(db.Connect(""), IsNil)
	c.Assert(db.InitDB(d.dbName), IsNil)
}
func (d *DB) TearDownSuite(c *C) {
	c.Assert(db.Exec(r.DBDrop(d.dbName)), IsNil)
}

func (d *DB) SetUpTest(c *C) {
	Clients.Clear()
	conf := config.ServerConfigs{}
	conf.Boards.Enabled = []string{"a"}
	config.Set(conf)
	for _, table := range db.AllTables {
		c.Assert(db.Write(r.Table(table).Delete()), IsNil)
	}
}

func syncAssertMessage(conn *websocket.Conn, msg []byte, c *C) {
	typ, buf, err := conn.ReadMessage()
	c.Assert(err, IsNil)
	c.Assert(typ, Equals, websocket.TextMessage)
	c.Assert(buf, DeepEquals, msg)
}

func marshalJSON(msg interface{}, c *C) []byte {
	data, err := json.Marshal(msg)
	c.Assert(err, IsNil)
	return data
}

func (*ClientSuite) TestDecodeMessage(c *C) {
	// Unparsable message
	var msg syncRequest
	err := decodeMessage([]byte{0}, &msg)
	c.Assert(err, ErrorMatches, "Invalid message structure")

	// Valid message
	std := syncRequest{
		Ctr:    5,
		Thread: 20,
		Board:  "a",
	}
	data := marshalJSON(std, c)
	c.Assert(decodeMessage(data, &msg), IsNil)
	c.Assert(msg, DeepEquals, std)
}
