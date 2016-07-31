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

	// Add client
	cl, _ := sv.NewClient()
	m.Add(cl, "1")
	c.Assert(m.clients[cl], Equals, "1")

	// Remove client
	m.Remove(cl)
	_, ok := m.clients[cl]
	c.Assert(ok, Equals, false)
}

func newClientMap() *ClientMap {
	return &ClientMap{
		clients: make(map[*Client]string),
	}
}

func (*Map) TestChangeSync(c *C) {
	const (
		oldThread = "1"
		newThread = "2"
	)
	m := newClientMap()
	sv := newWSServer(c)
	defer sv.Close()

	cl, _ := sv.NewClient()
	m.Add(cl, oldThread)
	c.Assert(m.clients[cl], Equals, oldThread)

	m.ChangeSync(cl, newThread)
	c.Assert(m.clients[cl], Equals, newThread)
}

func (*Map) TestCountByIP(c *C) {
	m := newClientMap()
	sv := newWSServer(c)
	defer sv.Close()

	cls := [3]*Client{}
	for i := range cls {
		cl, _ := sv.NewClient()
		cls[i] = cl
		m.Add(cl, "1")
	}
	cls[0].IP = "foo"
	cls[1].IP = "foo"
	cls[2].IP = "bar"

	c.Assert(m.CountByIP(), Equals, 2)
}
