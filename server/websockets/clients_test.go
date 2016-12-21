package websockets

import (
	"testing"

	. "github.com/bakape/meguca/test"
)

func newClientMap() *ClientMap {
	return &ClientMap{
		clients: make(map[*Client]SyncID),
		ips:     make(map[string]int),
	}
}

func TestMapAddHasRemove(t *testing.T) {
	t.Parallel()

	m := newClientMap()
	sv := newWSServer(t)
	defer sv.Close()
	id := SyncID{
		OP:    1,
		Board: "a",
	}

	// Add client
	cl, _ := sv.NewClient()
	m.add(cl, id)
	assertSyncID(t, m, cl, id)
	// Remove client
	m.remove(cl)
	if synced, _ := m.GetSync(cl); synced {
		t.Error("client still synced")
	}
}

func assertSyncID(t *testing.T, m *ClientMap, cl *Client, id SyncID) {
	synced, sync := m.GetSync(cl)
	if !synced {
		t.Error("client not synced")
	}
	if sync != id {
		LogUnexpected(t, id, sync)
	}
}

func TestMapChangeSync(t *testing.T) {
	t.Parallel()

	oldSync := SyncID{
		OP:    1,
		Board: "a",
	}
	newSync := SyncID{
		OP:    2,
		Board: "g",
	}
	m := newClientMap()
	sv := newWSServer(t)
	defer sv.Close()

	cl, _ := sv.NewClient()
	m.add(cl, oldSync)
	assertSyncID(t, m, cl, oldSync)

	m.changeSync(cl, newSync)
	assertSyncID(t, m, cl, newSync)
}

func TestCountByIP(t *testing.T) {
	t.Parallel()

	m := newClientMap()
	sv := newWSServer(t)
	defer sv.Close()

	cls := [3]*Client{}
	id := SyncID{
		OP:    1,
		Board: "a",
	}
	for i, ip := range [...]string{"foo", "foo", "bar"} {
		cl, _ := sv.NewClient()
		cl.ip = ip
		cls[i] = cl
		m.add(cl, id)
	}

	if count := len(m.ips); count != 2 {
		LogUnexpected(t, 2, count)
	}
}
