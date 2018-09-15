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
		`30{"recent":[1],"banned":[],"deleted":[],"deletedImage":[],"meidoVision":[],"open":{}}`,
	)

	assertMessage(t, wcl, "33[\"35{\\\"active\\\":0,\\\"total\\\":1}\"]")

	// Send message
	feeds.SendTo(1, []byte("foo"))
	assertMessage(t, wcl, "33[\"foo\"]")

	cl.Close(nil)
	sv.Wait()
}
