package websockets

import (
	"encoding/json"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/types"
	"github.com/bakape/meguca/util"
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

func (*ClientSuite) TestOldFeedClosing(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()

	closer := new(util.AtomicCloser)
	cl.updateFeedCloser = closer
	sv.Add(1)
	go func() {
		defer sv.Done()
		for closer.IsOpen() {
		}
	}()
	synchronise(nil, cl)
	sv.Wait()
	c.Assert(cl.updateFeedCloser, IsNil)
}

func (*ClientSuite) TestSyncToBoard(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	conf := config.ServerConfigs{}
	conf.Boards.Enabled = []string{"a"}
	config.Set(conf)

	// Invalid message
	c.Assert(synchronise(nil, cl), Equals, errInvalidStructure)

	// Invalid board
	msg := syncMessage{
		Thread: 0,
		Board:  "c",
	}
	data := marshalJSON(msg, c)
	c.Assert(synchronise(data, cl), Equals, errInvalidBoard)

	// Valid synchronisation
	msg.Board = "a"
	data = marshalJSON(msg, c)
	sv.Add(1)
	cl.ID = "hex"
	go assertMessage(wcl, []byte(`30{id:"hex"}`), sv, c)
	c.Assert(synchronise(data, cl), IsNil)
	sv.Wait()
}

func (*ClientSuite) TestRegisterSync(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()

	// Not synced yet
	registerSync("1", cl)
	id := cl.ID
	c.Assert(Clients.Has(id), Equals, true)
	c.Assert(Clients.clients[cl.ID].syncID, Equals, "1")

	// Already synced
	registerSync("2", cl)
	c.Assert(Clients.Has(id), Equals, true)
	c.Assert(Clients.clients[cl.ID].syncID, Equals, "2")
}

func (*DB) TestInvalidThreadSync(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()
	msg := syncMessage{
		Board:  "a",
		Thread: 1,
	}
	data := marshalJSON(msg, c)
	c.Assert(synchronise(data, cl), Equals, errInvalidThread)
}

func (*DB) TestSyncToThread(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	sv.Add(1)
	go readListenErrors(c, cl, sv)
	msg := syncMessage{
		Board:  "a",
		Thread: 1,
	}
	data := marshalJSON(msg, c)
	backlog1 := []byte{1, 2, 3}
	backlog2 := []byte{4, 5, 6}
	thread := types.DatabaseThread{
		ID:    1,
		Board: "a",
		Log:   [][]byte{backlog1, backlog2},
	}
	c.Assert(db.Write(r.Table("threads").Insert(thread)), IsNil)
	c.Assert(synchronise(data, cl), IsNil)
	c.Assert(Clients.Has(cl.ID), Equals, true)
	c.Assert(Clients.clients[cl.ID].syncID, Equals, "1")

	assertSyncResponse(wcl, cl, c)      // Receive client ID
	syncAssertMessage(wcl, backlog1, c) // Receive first missed message
	syncAssertMessage(wcl, backlog2, c) // Second message

	// Receive new message
	newMessage := []byte{7, 8, 9}
	update := map[string]r.Term{
		"log": r.Row.Field("log").Append(newMessage),
	}
	c.Assert(db.Write(r.Table("threads").Get(1).Update(update)), IsNil)
	syncAssertMessage(wcl, newMessage, c)
	cl.Close(nil)
	sv.Wait()
}

func assertSyncResponse(wcl *websocket.Conn, cl *Client, c *C) {
	res, err := encodeMessage(messageSynchronise, cl.ID)
	c.Assert(err, IsNil)
	syncAssertMessage(wcl, res, c)
}

// Test that only missed messages get sent as backlog.
func (*DB) TestOnlyMissedMessageSyncing(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	sv.Add(1)
	go readListenErrors(c, cl, sv)

	msg := syncMessage{
		Board:  "a",
		Thread: 1,
		Ctr:    1,
	}
	data := marshalJSON(msg, c)
	backlogs := [][]byte{
		{1, 2, 3},
		{4, 5, 6},
		{7, 8, 9},
	}
	thread := types.DatabaseThread{
		ID:    1,
		Board: "a",
		Log:   backlogs,
	}
	c.Assert(db.Write(r.Table("threads").Insert(thread)), IsNil)

	c.Assert(synchronise(data, cl), IsNil)
	assertSyncResponse(wcl, cl, c)         // Receive client ID
	syncAssertMessage(wcl, backlogs[1], c) // Receive first missed message
	syncAssertMessage(wcl, backlogs[2], c) // Second missed message
	cl.Close(nil)
	sv.Wait()
}

func (*DB) TestMaliciousCounterGuard(c *C) {
	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()
	thread := types.DatabaseThread{
		ID:    1,
		Board: "a",
		Log:   [][]byte{{1}},
	}
	c.Assert(db.Write(r.Table("threads").Insert(thread)), IsNil)

	// Negative counter
	msg := syncMessage{
		Board:  "a",
		Thread: 1,
		Ctr:    -10,
	}
	data := marshalJSON(msg, c)
	c.Assert(synchronise(data, cl), Equals, errInvalidCounter)

	// Counter larger than in the database
	msg.Ctr = 7
	data = marshalJSON(msg, c)
	c.Assert(synchronise(data, cl), Equals, errInvalidCounter)
}

func (*DB) TestRegistrationStringValidations(c *C) {
	samples := []struct {
		id, password string
		code         accountResponse
	}{
		{"12", "123456", idTooShort},
		{util.RandomID(21), "123456", idTooLong},
		{"123", "12345", passwordTooShort},
		{"123", util.RandomID(31), passwordTooLong},
	}

	for _, s := range samples {
		code, err := handleRegistration(s.id, s.password)
		c.Assert(err, IsNil)
		c.Assert(code, Equals, s.code)
	}
}

func (*DB) TestRegistration(c *C) {
	req := registrationRequest{
		ID:       "123",
		Password: "123456",
	}
	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	data := marshalJSON(req, c)

	// Valid registration
	sv.Add(1)
	c.Assert(register(data, cl), IsNil)
	msg, err := encodeMessage(messageLogin, loginSuccess)
	c.Assert(err, IsNil)
	c.Assert(cl.loggedIn, Equals, true)
	assertMessage(wcl, msg, sv, c)
	sv.Wait()

	// User name taken
	cl, wcl = sv.NewClient()
	sv.Add(1)
	c.Assert(register(data, cl), IsNil)
	msg, err = encodeMessage(messageLogin, userNameTaken)
	c.Assert(err, IsNil)
	c.Assert(cl.loggedIn, Equals, false)
	assertMessage(wcl, msg, sv, c)
	sv.Wait()
}

func (*DB) TestAlreadyLoggedIn(c *C) {
	cl := &Client{
		loggedIn: true,
	}
	c.Assert(register(nil, cl), ErrorMatches, "already logged in")
}
