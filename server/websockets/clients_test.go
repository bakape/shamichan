package websockets

import (
	. "gopkg.in/check.v1"
)

type Map struct{}

var _ = Suite(&Map{})

func (*Map) TestAddHasRemove(c *C) {
	m := newClientMap()
	sv := newWSServer(c)
	defer sv.Close()

	// Add client
	cl, _ := sv.NewClient()
	m.Add(cl, "1")
	c.Assert(cl.ID, Matches, "[A-Za-z0-9]{16}")
	c.Assert(m.Has(cl.ID), Equals, true)

	// Remove client
	m.Remove(cl.ID)
	c.Assert(m.Has(cl.ID), Equals, false)
}

func newClientMap() *ClientMap {
	return &ClientMap{
		clients: make(map[string]clientContainer),
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
	c.Assert(m.clients[cl.ID].syncID, Equals, oldThread)

	m.ChangeSync(cl.ID, newThread)
	c.Assert(m.clients[cl.ID].syncID, Equals, newThread)
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
	cls[0].ident.IP = "foo"
	cls[1].ident.IP = "foo"
	cls[2].ident.IP = "bar"

	c.Assert(m.CountByIP(), Equals, 2)
}

func (*Map) TestSendAll(c *C) {
	m := newClientMap()
	sv := newWSServer(c)
	defer sv.Close()
	msg := []byte{1, 2, 3}

	cls := [3]*Client{}
	for i := range cls {
		cl, _ := sv.NewClient()
		cls[i] = cl
		m.Add(cl, "1")
		sv.Add(1)
		go func() {
			c.Assert(<-cl.Send, DeepEquals, msg)
			sv.Done()
		}()
	}

	m.SendAll(msg)
	sv.Wait()
}
