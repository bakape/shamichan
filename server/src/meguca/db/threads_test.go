package db

import (
	"database/sql"
	"meguca/common"
	"meguca/test"
	"testing"
	"time"
)

func TestThreadBools(t *testing.T) {
	assertTableClear(t, "boards")
	writeSampleBoard(t)
	writeSampleThread(t)

	locked, err := CheckThreadLocked(1)
	if err != nil {
		t.Fatal(err)
	}
	test.AssertDeepEquals(t, false, locked)
}

func TestDiffPostCount(t *testing.T) {
	// Reset state
	postCountCacheMu.Lock()
	postCountCache = make(map[uint64]uint64)
	postCountCacheMu.Unlock()

	prepareThreads(t)

	init := map[uint64]uint64{
		1: 0,
		3: 1,
		4: 6,
	}
	std := ThreadPostCountDiff{
		Changed: map[uint64]int{
			1: 3,
		},
		Deleted: []uint64{4},
	}

	assert := func() {
		// Sleep to ensure notifications fire
		time.Sleep(time.Millisecond * 100)

		res, err := DiffThreadPostCounts(init)
		if err != nil {
			t.Fatal(err)
		}
		test.AssertDeepEquals(t, res, std)
	}

	assert()

	_, err := sq.Delete("threads").
		Where("id = 3").
		Exec()
	if err != nil {
		t.Fatal(err)
	}
	// Only allow one deleted to avoid mao reordering issues
	delete(init, 4)
	std.Deleted[0] = 3
	assert()

	err = InTransaction(false, func(tx *sql.Tx) error {
		return WritePost(
			tx,
			Post{
				StandalonePost: common.StandalonePost{
					Post: common.Post{
						ID:   7,
						Time: time.Now().Unix(),
					},
					OP:    1,
					Board: "a",
				},
				IP: "::1",
			},
			true,
			false)
	})
	if err != nil {
		t.Fatal(err)
	}

	std.Changed[1] = 4
	assert()
}
