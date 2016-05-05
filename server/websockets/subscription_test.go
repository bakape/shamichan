package websockets

import (
	"github.com/bakape/meguca/db"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
	"time"
)

type SubSuite struct {
	dbName string
}

var _ = Suite(&SubSuite{})

func (s *SubSuite) SetUpSuite(c *C) {
	s.dbName = db.UniqueDBName()
	c.Assert(db.Connect(""), IsNil)
	c.Assert(db.InitDB(s.dbName), IsNil)
}

func (s *SubSuite) TearDownSuite(c *C) {
	c.Assert(db.DB(r.DBDrop(s.dbName)).Exec(), IsNil)
	c.Assert(db.RSession.Close(), IsNil)
}

// Clear all documents from all tables after each test.
func (*SubSuite) TearDownTest(c *C) {
	Subs.subs = make(map[int64]*Subscription)
	for _, table := range db.AllTables {
		c.Assert(db.DB(r.Table(table).Delete()).Exec(), IsNil)
	}
}

func (*SubSuite) TestBasicSubscription(c *C) {
	const id = 10032
	sv := newWSServer(c)
	defer sv.Close()
	std := []byte{2, 0, 0, 0, 1}
	thread := map[string]interface{}{
		"id":  id,
		"log": []string{"log"},
	}
	c.Assert(db.DB(r.Table("threads").Insert(thread)).Exec(), IsNil)

	// New Subscription
	c.Assert(Subs.Exists(id), Equals, false)
	cl, _ := sv.NewClient()
	Clients.Add(cl)
	Subs.ListenTo(id, cl)
	c.Assert(Subs.Exists(id), Equals, true)
	sv.Add(1)
	go assertMessage(c, cl, std, sv)
	Subs.subs[id].write <- std
	sv.Wait()

	// Existing subscription
	cl2, _ := sv.NewClient()
	Clients.Add(cl2)
	Subs.ListenTo(id, cl2)
	sv.Add(2)
	go assertMessage(c, cl, std, sv)
	go assertMessage(c, cl2, std, sv)
	Subs.subs[id].write <- std
	sv.Wait()

	// Clean up on no clients
	Subs.Unlisten(id, cl.ID)
	Subs.Unlisten(id, cl2.ID)
	time.Sleep(time.Second * 15) // Wait out shutdown timer
	c.Assert(Subs.Exists(id), Equals, false)
}

func assertMessage(c *C, cl *Client, std []byte, sv *mockWSServer) {
	defer sv.Done()
	c.Assert(<-cl.sender, DeepEquals, std)
}
