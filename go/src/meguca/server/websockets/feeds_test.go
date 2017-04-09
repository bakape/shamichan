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
	sv.Add(2)
	cl1, wcl1 := sv.NewClient()
	addToFeed(t, cl1, 1)
	go readListenErrors(t, cl1, sv)
	cl2, wcl2 := sv.NewClient()
	addToFeed(t, cl2, 1)
	go readListenErrors(t, cl2, sv)

	const syncMsg = "30{\"recent\":[1],\"open\":{\"1\":{\"hasImage\":false,\"body\":\"\"}}}"
	assertMessage(t, wcl1, syncMsg)
	assertMessage(t, wcl2, syncMsg)

	// Send message
	feeds.SendTo(1, []byte("foo"))
	assertMessage(t, wcl1, "33foo")
	assertMessage(t, wcl2, "33foo")

	cl1.Close(nil)
	cl2.Close(nil)
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

func TestFlushMultipleMessages(t *testing.T) {
	t.Parallel()

	sv := newWSServer(t)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	sv.Add(1)
	go readListenErrors(t, cl, sv)
	const msg = "a\u0000bc"
	u := updateFeed{
		clients: []*Client{cl},
	}
	u.Write([]byte("a"))
	u.Write([]byte("bc"))

	u.flushBuffer()
	assertMessage(t, wcl, encodeMessageType(common.MessageConcat)+msg)

	cl.Close(nil)
	sv.Wait()
}

func encodeMessageType(typ common.MessageType) string {
	return strconv.Itoa(int(typ))
}
