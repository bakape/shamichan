package websockets

import (
	"testing"
)

func newClientMap() *ClientMap {
	return &ClientMap{
		clients: make(map[*Client]SyncID),
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
	m.Add(cl, id)
	assertSyncID(t, m, cl, id)
	// Remove client
	m.Remove(cl)
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
		logUnexpected(t, id, sync)
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
	m.Add(cl, oldSync)
	assertSyncID(t, m, cl, oldSync)

	m.ChangeSync(cl, newSync)
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
	for i := range cls {
		cl, _ := sv.NewClient()
		cls[i] = cl
		m.Add(cl, id)
	}
	cls[0].IP = "foo"
	cls[1].IP = "foo"
	cls[2].IP = "bar"

	if count := m.CountByIP(); count != 2 {
		logUnexpected(t, 2, count)
	}
}
