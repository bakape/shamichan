package websockets

import (
	"encoding/json"
	. "gopkg.in/check.v1"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/types"
	"github.com/bakape/meguca/db"
	r "github.com/dancannon/gorethink"
)

var _ = Suite(&DB{})

// Tests that require database access
type DB struct{
	dbName string
}

func (d *DB) SetUpSuite(c *C) {
	d.dbName = db.UniqueDBName()
	c.Assert(db.Connect(""), IsNil)
	c.Assert(db.InitDB(d.dbName), IsNil)
}

func (d *DB) SetUpTest(c *C)  {
	Clients.Clear()
	conf := config.ServerConfigs{}
	conf.Boards.Enabled = []string{"a"}
	config.Set(conf)
	for _, table := range db.AllTables {
		c.Assert(db.DB(r.Table(table).Delete()).Exec(), IsNil)
	}
}

func (*ClientSuite) TestDecodeMessage(c *C) {
	// Unparsable message
	var msg syncMessage
	err := decodeMessage([]byte{0}, &msg)
	c.Assert(err, ErrorMatches, "Invalid message structure")

	// Valid message
	std := syncMessage{
		Ctr:    5,
		Thread: 20,
		Board:  "a",
	}
	data := marshalJSON(std, c)
	c.Assert(decodeMessage(data, &msg), IsNil)
	c.Assert(msg, DeepEquals, std)
}

func marshalJSON(msg interface{}, c *C) []byte {
	data, err := json.Marshal(msg)
	c.Assert(err, IsNil)
	return data
}

func (*ClientSuite) TestOldFeedClosing(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()

	closeFeed := make(chan struct{})
	cl.closeFeed = closeFeed
	sv.Add(1)
	go func() {
		defer sv.Done()
		<-closeFeed
	}()
	cl.synchronise(nil)
	sv.Wait()
	c.Assert(cl.closeFeed, IsNil)
}

func (*ClientSuite) TestSyncToBoard(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	conf := config.ServerConfigs{}
	conf.Boards.Enabled = []string{"a"}
	config.Set(conf)

	// Invalid message
	c.Assert(cl.synchronise(nil), Equals, errInvalidStructure)

	// Invalid board
	msg := syncMessage{
		Thread: 0,
		Board: "c",
	}
	data := marshalJSON(msg, c)
	c.Assert(cl.synchronise(data), Equals, errInvalidBoard)

	// Valid synchronisation
	msg.Board = "a"
	data = marshalJSON(msg, c)
	sv.Add(1)
	cl.ID = "hex"
	go assertMessage(wcl, []byte(`30{id:"hex"}`), sv, c)
	c.Assert(cl.synchronise(data), IsNil)
	sv.Wait()
}

func (*ClientSuite) TestRegisterSync(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()

	// Not synced yet
	cl.registerSync("1")
	id := cl.ID
	c.Assert(Clients.Has(id), Equals, true)
	c.Assert(Clients.clients[cl.ID].syncID, Equals, "1")

	// Already synced
	cl.registerSync("2")
	c.Assert(Clients.Has(id), Equals, true)
	c.Assert(Clients.clients[cl.ID].syncID, Equals, "2")
}


func (*DB) TestSyncToThread(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	msg := syncMessage{
		Board: "a",
		Thread: 1,
	}

	// Invalid thread in request
	data := marshalJSON(msg, c)
	c.Assert(cl.synchronise(data), Equals, errInvalidThread)

	backlog1 := []byte{1,2,3}
	backlog2 := []byte{4,5,6}
	thread := types.DatabaseThread{
		ID: 1,
		Board: "a",
		Log: [][]byte{backlog1, backlog2},
	}
	c.Assert(db.DB(r.Table("threads").Insert(thread)).Exec(), IsNil)

	// Receive missed messages
	sv.Add(1)
	go cl.Listen()
	go assertMessage(wcl, backlog1, sv, c)
	c.Assert(cl.synchronise(data), IsNil)
	c.Assert(Clients.Has(cl.ID), Equals, true)
	c.Assert(Clients.clients[cl.ID].syncID, Equals, "1")
	sv.Wait()

	// Second message
	sv.Add(1)
	go assertMessage(wcl, backlog2, sv, c)
	sv.Wait()

	// Receive new messages
	newMessage := []byte{7,8,9}
	update := map[string]r.Term{
		"log": r.Row.Field("log").Append(newMessage),
	}
	c.Assert(db.DB(r.Table("threads").Get(1).Update(update)).Exec(), IsNil)

	sv.Add(1)
	go assertMessage(wcl, newMessage, sv, c)
	sv.Wait()
	closeClient(c, cl)

	// Test that only missed messages get sent as backlog
	cl, wcl = sv.NewClient()
	msg.Ctr = 1
	data = marshalJSON(msg, c)
	sv.Add(1)
	go cl.Listen()
	go assertMessage(wcl, backlog2, sv, c)
	c.Assert(cl.synchronise(data), IsNil)
	sv.Wait()

	sv.Add(1)
	go assertMessage(wcl, newMessage, sv, c)
	sv.Wait()
	closeClient(c, cl)
}

func (*DB) TestMaliciousCounterGuard(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()
	thread := types.DatabaseThread{
		ID: 1,
		Board: "a",
		Log: [][]byte{{1}},
	}
	c.Assert(db.DB(r.Table("threads").Insert(thread)).Exec(), IsNil)

	// Negative counter
	msg := syncMessage{
		Board: "a",
		Thread: 1,
		Ctr: -10,
	}
	data := marshalJSON(msg, c)
	c.Assert(cl.synchronise(data), Equals, errInvalidCounter)

	// Counter larger than in the database
	msg.Ctr = 7
	data = marshalJSON(msg, c)
	c.Assert(cl.synchronise(data), Equals, errInvalidCounter)
}
