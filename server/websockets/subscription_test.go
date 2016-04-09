package websockets

import (
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
	"strconv"
	"time"
)

type SubSuite struct {
	dbName string
}

var _ = Suite(&SubSuite{})

func (s *SubSuite) SetUpSuite(c *C) {
	s.dbName = "meguca_tests_" + strconv.FormatInt(time.Now().UnixNano(), 10)

	var err error
	db.RSession, err = r.Connect(r.ConnectOpts{
		Address: "localhost:28015",
	})
	c.Assert(err, IsNil)

	c.Assert(db.DB(r.DBCreate(s.dbName)).Exec(), IsNil)
	db.RSession.Use(s.dbName)
	c.Assert(db.CreateTables(), IsNil)
	c.Assert(db.CreateIndeces(), IsNil)
}

func (s *SubSuite) TearDownSuite(c *C) {
	c.Assert(r.DBDrop(s.dbName).Exec(db.RSession), IsNil)
	c.Assert(db.RSession.Close(), IsNil)
}

// Clear all documents from all tables after each test.
func (*SubSuite) TearDownTest(c *C) {
	Subs.subs = make(map[uint64]*Subscription)
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

func (*SubSuite) TestReadJSON(c *C) {
	// No thread
	c.Assert(Subs.ThreadJSON(1), IsNil)

	thread := map[string]interface{}{
		"id":     1,
		"logCtr": 10,
		"log":    []string{"log"},
	}
	c.Assert(db.DB(r.Table("threads").Insert(thread)).Exec(), IsNil)
	post := types.Post{ID: 1}
	c.Assert(db.DB(r.Table("posts").Insert(post)).Exec(), IsNil)

	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()
	Clients.Add(cl)
	Subs.ListenTo(1, cl)
	defer Subs.Unlisten(1, cl.ID)

	// No cached data
	sub := Subs.subs[1]
	sub.counter = 20
	c.Assert(Subs.ThreadJSON(1), NotNil)
	c.Assert(sub.data.counter, Equals, uint64(10))

	// Cached data still fresh
	std := []byte{1, 2, 3}
	sub.data.buffer = std
	sub.data.counter = 20
	c.Assert(Subs.ThreadJSON(1), DeepEquals, std)
}
