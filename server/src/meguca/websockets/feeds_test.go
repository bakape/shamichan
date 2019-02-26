package websockets

import (
	"meguca/test/test_db"
	"meguca/websockets/feeds"
	"testing"
)

func TestStreamUpdates(t *testing.T) {
	feeds.Clear()
	test_db.ClearTables(t, "boards")
	test_db.WriteSampleBoard(t)
	test_db.WriteSampleThread(t)

	sv := newWSServer(t)
	defer sv.Close()
	sv.Add(1)
	cl, wcl := sv.NewClient()
	registerClient(t, cl, 1, "a")
	go readListenErrors(t, cl, sv)

	assertMessage(t, wcl, `30{"recent":{},"moderation":{}}`)
	assertMessage(t, wcl, "33[\"35{\\\"active\\\":0,\\\"total\\\":1}\"]")

	// Send message
	feeds.SendTo(1, []byte("foo"))
	assertMessage(t, wcl, "33[\"foo\"]")

	cl.Close(nil)
	sv.Wait()
}
