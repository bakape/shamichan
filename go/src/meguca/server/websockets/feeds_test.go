package websockets

import (
	"meguca/common"
	. "meguca/test"
	"strconv"
	"testing"

	"github.com/gorilla/websocket"
)

func TestStreamUpdates(t *testing.T) {
	feeds.Clear()
	assertTableClear(t, "boards")
	writeSampleBoard(t)
	writeSampleThread(t)

	sv := newWSServer(t)
	defer sv.Close()
	sv.Add(2)
	var (
		cls  [2]*Client
		wcls [2]*websocket.Conn
	)
	for i := range cls {
		cls[i], wcls[i] = sv.NewClient()
		cls[i].ip = strconv.Itoa(i)
		addToFeed(t, cls[i], 1)
		go readListenErrors(t, cls[i], sv)
	}

	const syncMsg = "30{\"recent\":[1],\"open\":{\"1\":{\"hasImage\":false,\"body\":\"\"}}}"
	assertMessage(t, wcls[0], syncMsg)
	assertMessage(t, wcls[0], "351")
	assertMessage(t, wcls[1], syncMsg)
	for i := range wcls {
		assertMessage(t, wcls[i], "352")
	}

	// Send message
	feeds.SendTo(1, []byte("foo"))
	for i := range wcls {
		assertMessage(t, wcls[i], "33foo")
	}

	for i := range cls {
		cls[i].Close(nil)
	}
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
