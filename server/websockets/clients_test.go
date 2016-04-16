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
	m.Add(cl)
	c.Assert(m.Has(cl.ID), Equals, true)

	// Remove client
	m.Remove(cl.ID)
	c.Assert(m.Has(cl.ID), Equals, false)
}

func newClientMap() *ClientMap {
	return &ClientMap{
		clients: make(map[string]*Client),
	}
}

func (*Map) TestCountByIP(c *C) {
	m := newClientMap()
	sv := newWSServer(c)
	defer sv.Close()

	cls := [3]*Client{}
	for i := range cls {
		cl, _ := sv.NewClient()
		cls[i] = cl
		m.Add(cl)
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
		m.Add(cl)
		sv.Add(1)
		go assertMessage(c, cl, msg, sv)
	}

	m.SendAll(msg)
	sv.Wait()
}
