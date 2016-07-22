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
	c.Assert(cl.ID, Matches, "^.{43}$")
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
	cls[0].IP = "foo"
	cls[1].IP = "foo"
	cls[2].IP = "bar"

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

func (*Map) TestGetNonExistantClient(c *C) {
	m := newClientMap()
	_, err := m.Get("1")
	c.Assert(err, ErrorMatches, "no client found: .*")
}

func (*Map) TestGetClient(c *C) {
	m := newClientMap()
	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()
	m.Add(cl, "100")

	res, err := m.Get(cl.ID)
	c.Assert(err, IsNil)
	c.Assert(res, DeepEquals, cl)
}
