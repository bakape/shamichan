package websockets

import (
	"meguca/common"
	. "meguca/test"
	"strconv"
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
	addToFeed(t, cl, 1)
	go readListenErrors(t, cl, sv)

	assertMessage(
		t,
		wcl,
		`30{"recent":[1],"open":{"1":{"hasImage":false,"body":""}}}`,
	)
	assertMessage(t, wcl, "33351")

	// Send message
	feeds.SendTo(1, []byte("foo"))
	assertMessage(t, wcl, "33foo")

	cl.Close(nil)
	sv.Wait()
}

func TestWriteMultipleToBuffer(t *testing.T) {
	t.Parallel()

	u := updateFeed{}
	u.Write([]byte("a"))
	u.Write([]byte("b"))

	const std = "33a\u0000b"
	if s := string(u.Flush()); s != std {
		LogUnexpected(t, std, s)
	}
}

func encodeMessageType(typ common.MessageType) string {
	return strconv.Itoa(int(typ))
}
