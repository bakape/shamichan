package feeds

import (
	"database/sql"
	"meguca/common"
	"meguca/config"
	"meguca/db"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-playground/log"
	"github.com/go-playground/log/handlers/console"
)

func init() {
	log.AddHandler(console.New(true), log.AllLevels...)
	db.ConnArgs = db.TestConnArgs
	db.IsTest = true
	if err := db.LoadDB(); err != nil {
		panic(err)
	}
}

type dummyClient struct{}

func (d *dummyClient) Send(_ []byte)     {}
func (d *dummyClient) Redirect(_ string) {}
func (d *dummyClient) IP() string        { return "::1" }
func (d *dummyClient) NewProtocol() bool { return false }
func (d *dummyClient) Last100() bool     { return false }
func (d *dummyClient) Close(_ error)     {}

func TestThreadWatcher(t *testing.T) {
	assertTableClear(t, "boards")
	writeSampleBoard(t)
	writeSampleThread(t)
	Clear()
	terminateWatcher = make(chan bool)

	r1 := httptest.NewRequest("GET", "/", nil)
	rec1 := httptest.NewRecorder()
	var wg sync.WaitGroup

	// First send request to wait for feeds
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := WatchThreads(rec1, r1, []uint64{1, 2, 3})
		if err != nil {
			t.Fatal(err)
		}
	}()

	// To more or less make sure the async stuff actually happens
	time.Sleep(time.Second)

	// Then migrate watcher to feed
	c1 := new(dummyClient)
	feed, err := SyncClient(c1, 1, "a")
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second)

	// Add new watcher to existing feed
	r2 := httptest.NewRequest("GET", "/", nil)
	rec2 := httptest.NewRecorder()
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := WatchThreads(rec2, r2, []uint64{1, 2, 3})
		if err != nil {
			t.Fatal(err)
		}
	}()

	// Send a message to both watchers

	std := common.Post{
		Body:    "foo",
		ID:      2,
		Editing: true,
	}
	feed.InsertPost(std, nil)
	std.Links = []common.Link{
		{
			ID:    22,
			OP:    22,
			Board: "c",
		},
	}
	feed.ClosePost(std.ID, std.Links, nil, nil)

	// Can't test Keep-alive requests that easily, so just test assert this runs

	time.Sleep(time.Second)
	terminateWatcher <- true
	terminateWatcher <- true
	wg.Wait()
}

func assertTableClear(t testing.TB, tables ...string) {
	t.Helper()

	if err := db.ClearTables(tables...); err != nil {
		t.Fatal(err)
	}
}
func writeSampleBoard(t testing.TB) {
	t.Helper()

	b := db.BoardConfigs{
		BoardConfigs: config.BoardConfigs{
			ID:        "a",
			Eightball: []string{"yes"},
		},
	}
	err := db.InTransaction(false, func(tx *sql.Tx) error {
		return db.WriteBoard(tx, b)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func writeSampleThread(t testing.TB) {
	t.Helper()

	now := time.Now().Unix()
	thread := db.Thread{
		ID:        1,
		Board:     "a",
		PostCtr:   0,
		ImageCtr:  1,
		ReplyTime: now,
	}
	op := db.Post{
		StandalonePost: common.StandalonePost{
			Post: common.Post{
				ID:   1,
				Time: time.Now().Unix(),
			},
			OP: 1,
		},
	}
	if err := db.WriteThread(nil, thread, op); err != nil {
		t.Fatal(err)
	}
}
