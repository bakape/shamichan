package websockets

import (
	"strconv"
	"testing"
	"time"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/db"
	. "github.com/bakape/meguca/test"
)

func TestAddingFeeds(t *testing.T) {
	assertTableClear(t, "boards")
	writeSampleBoard(t)
	writeSampleThread(t)
	feeds.Clear()

	sv := newWSServer(t)
	defer sv.Close()
	sv.Add(2)
	cl1, wcl1 := sv.NewClient()
	go readListenErrors(t, cl1, sv)
	cl2, wcl2 := sv.NewClient()
	go readListenErrors(t, cl2, sv)

	if err := feeds.Add(1, cl1); err != nil {
		t.Fatal(err)
	}
	defer feeds.Clear()
	assertMessage(t, wcl1, "300")

	if err := feeds.Add(1, cl2); err != nil {
		t.Fatal(err)
	}
	assertMessage(t, wcl2, "300")

	feeds.Remove(1, cl2)

	cl1.Close(nil)
	cl2.Close(nil)
	sv.Wait()
}

func TestStreamUpdates(t *testing.T) {
	assertTableClear(t, "boards")
	writeSampleBoard(t)
	writeSampleThread(t)
	feeds.Clear()

	sv := newWSServer(t)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	sv.Add(2)
	go readListenErrors(t, cl, sv)
	if err := feeds.Add(1, cl); err != nil {
		t.Fatal(err)
	}
	defer feeds.Clear()

	assertMessage(t, wcl, "300")

	// One message
	if err := db.UpdateLog(1, []byte("foo")); err != nil {
		t.Fatal(err)
	}
	assertMessage(t, wcl, "33foo")

	// Another
	if err := db.UpdateLog(1, []byte("bar")); err != nil {
		t.Fatal(err)
	}
	assertMessage(t, wcl, "33bar")

	// Count updated
	time.Sleep(time.Millisecond * 200)
	cl2, wcl2 := sv.NewClient()
	go readListenErrors(t, cl2, sv)
	if err := feeds.Add(1, cl2); err != nil {
		t.Fatal(err)
	}
	assertMessage(t, wcl2, "302")

	cl.Close(nil)
	cl2.Close(nil)
	sv.Wait()
}

func encodeMessage(
	t *testing.T,
	typ common.MessageType,
	data interface{},
) string {
	msg, err := common.EncodeMessage(typ, data)
	if err != nil {
		t.Fatal(err)
	}
	return string(msg)
}

func TestWriteMultipleToBuffer(t *testing.T) {
	t.Parallel()

	u := updateFeed{}
	u.Write([]byte("a"))
	u.Write([]byte("b"))

	const std = "33a\u0000b"
	buf, flushed := u.Flush()
	if s := string(buf); s != std {
		LogUnexpected(t, std, s)
	}
	if flushed != 2 {
		t.Fatalf("unexpected message count: %d", flushed)
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
