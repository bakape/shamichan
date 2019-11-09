package db

import (
	"database/sql"
	"testing"
	"time"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/test"
)

func TestThreadBools(t *testing.T) {
	assertTableClear(t, "boards")
	writeSampleBoard(t)
	writeSampleThread(t)

	locked, err := CheckThreadLocked(1)
	if err != nil {
		t.Fatal(err)
	}
	test.AssertEquals(t, false, locked)
}

func TestDiffPostCount(t *testing.T) {
	// Reset state
	postCountCacheMu.Lock()
	postCountCache = make(map[uint64]uint64)
	postCountCacheMu.Unlock()
	prepareThreads(t)
	err := loadThreadPostCounts()
	if err != nil {
		t.Fatal(err)
	}

	canceller := make(chan struct{})
	defer func() {
		canceller <- struct{}{}
	}()
	err = listenForThreadUpdates(canceller)
	if err != nil {
		t.Fatal(err)
	}

	init := map[uint64]uint64{
		1: 0,
		3: 1,
		4: 6,
	}
	std := ThreadPostCountDiff{
		Changed: map[uint64]uint64{
			1: 109,
		},
		Deleted: []uint64{4},
	}

	assert := func(t *testing.T) {
		t.Helper()

		// Sleep to ensure notifications fire
		time.Sleep(time.Millisecond * 100)

		res, err := DiffThreadPostCounts(init)
		if err != nil {
			t.Fatal(err)
		}
		test.AssertEquals(t, res, std)
	}

	assert(t)

	_, err = sq.Delete("threads").
		Where("id = 3").
		Exec()
	if err != nil {
		t.Fatal(err)
	}
	// Only allow one deleted to avoid mao reordering issues
	delete(init, 4)
	std.Deleted[0] = 3
	assert(t)

	err = InTransaction(func(tx *pgx.Tx) error {
		return WritePost(
			tx,
			Post{
				StandalonePost: common.StandalonePost{
					Post: common.Post{
						ID:   999,
						Time: time.Now().Unix(),
					},
					OP:    1,
					Board: "a",
				},
				IP: "::1",
			},
		)
	})
	if err != nil {
		t.Fatal(err)
	}

	std.Changed[1] = 110
	assert(t)
}

func TestInsertThread(t *testing.T) {
	assertTableClear(t, "boards")
	writeSampleBoard(t)

	p := Post{
		StandalonePost: common.StandalonePost{
			Board: "a",
		},
		IP:       "::1",
		Password: []byte("6+53653cs3ds"),
	}
	err := InTransaction(func(tx *pgx.Tx) (err error) {
		return InsertThread(tx, "test", &p)
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.Time == 0 {
		t.Fatal(p.Time)
	}
	if p.OP == 0 {
		t.Fatal(p.OP)
	}
	if p.ID == 0 {
		t.Fatal(p.ID)
	}
}
