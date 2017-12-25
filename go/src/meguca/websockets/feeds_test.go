package websockets

import (
	"meguca/websockets/feeds"
	"testing"
)

func TestStreamUpdates(t *testing.T) {
	feeds.Clear()
	assertTableClear(t, "boards")
	writeSampleBoard(t)
	writeSampleThread(t)

	sv := newWSServer(t)
	defer sv.Close()
	sv.Add(1)
	cl, wcl := sv.NewClient()
	registerClient(t, cl, 1, "a")
	go readListenErrors(t, cl, sv)

	assertMessage(
		t,
		wcl,
		`30{"replies":[],"banned":[],"deleted":[],"spoilered":[],"deletedImages":[]}`,
	)
	assertMessage(t, wcl, "32341")

	// Send message
	feeds.SendTo(1, []byte("foo"))
	assertMessage(t, wcl, "32foo")

	cl.Close(nil)
	sv.Wait()
}
