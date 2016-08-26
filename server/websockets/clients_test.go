package websockets

import (
	"github.com/bakape/meguca/config"
	. "gopkg.in/check.v1"
)

type Map struct{}

var _ = Suite(&Map{})

func (*Map) SetUpTest(_ *C) {
	config.Set(config.Configs{})
}

func (*Map) TestAddHasRemove(c *C) {
	m := newClientMap()
	sv := newWSServer(c)
	defer sv.Close()
	id := SyncID{
		OP:    1,
		Board: "a",
	}

	// Add client
	cl, _ := sv.NewClient()
	m.Add(cl, id)
	synced, sync := m.GetSync(cl)
	c.Assert(synced, Equals, true)
	c.Assert(sync, Equals, id)

	// Remove client
	m.Remove(cl)
	synced, _ = m.GetSync(cl)
	c.Assert(synced, Equals, false)
}

func newClientMap() *ClientMap {
	return &ClientMap{
		clients: make(map[*Client]SyncID),
	}
}

func (*Map) TestChangeSync(c *C) {
	oldSync := SyncID{
		OP:    1,
		Board: "a",
	}
	newSync := SyncID{
		OP:    2,
		Board: "g",
	}
	m := newClientMap()
	sv := newWSServer(c)
	defer sv.Close()

	cl, _ := sv.NewClient()
	m.Add(cl, oldSync)
	synced, sync := m.GetSync(cl)
	c.Assert(synced, Equals, true)
	c.Assert(sync, Equals, oldSync)

	m.ChangeSync(cl, newSync)
	synced, sync = m.GetSync(cl)
	c.Assert(synced, Equals, true)
	c.Assert(sync, Equals, newSync)
}

func (*Map) TestCountByIP(c *C) {
	m := newClientMap()
	sv := newWSServer(c)
	defer sv.Close()

	cls := [3]*Client{}
	id := SyncID{
		OP:    1,
		Board: "a",
	}
	for i := range cls {
		cl, _ := sv.NewClient()
		cls[i] = cl
		m.Add(cl, id)
	}
	cls[0].IP = "foo"
	cls[1].IP = "foo"
	cls[2].IP = "bar"

	c.Assert(m.CountByIP(), Equals, 2)
}
