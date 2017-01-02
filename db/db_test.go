package db

import (
	"testing"

	"github.com/bakape/meguca/common"
	. "github.com/bakape/meguca/test"
)

func TestValidateOp(t *testing.T) {
	assertTableClear(t, "threads")
	assertInsert(t, "threads", common.DatabaseThread{
		ID:    1,
		Board: "a",
	})

	samples := [...]struct {
		id      uint64
		board   string
		isValid bool
	}{
		{1, "a", true},
		{15, "a", false},
	}

	for i := range samples {
		s := samples[i]
		t.Run("", func(t *testing.T) {
			t.Parallel()
			valid, err := ValidateOP(s.id, s.board)
			if err != nil {
				t.Fatal(err)
			}
			if valid != s.isValid {
				t.Fatal("unexpected result")
			}
		})
	}
}

func TestThreadCounter(t *testing.T) {
	assertTableClear(t, "posts")
	assertInsert(t, "posts", []common.DatabasePost{
		{
			StandalonePost: common.StandalonePost{
				OP: 1,
				Post: common.Post{
					ID: 1,
				},
			},
			LastUpdated: 54,
		},
		{
			StandalonePost: common.StandalonePost{
				OP: 1,
				Post: common.Post{
					ID: 2,
				},
			},
			LastUpdated: 55,
		},
	})

	ctr, err := ThreadCounter(1)
	if err != nil {
		t.Fatal(err)
	}
	if ctr != 55 {
		LogUnexpected(t, 55, ctr)
	}
}

func TestBoardCounter(t *testing.T) {
	assertTableClear(t, "posts")
	assertInsert(t, "posts", []common.DatabasePost{
		{
			StandalonePost: common.StandalonePost{
				Board: "a",
				Post: common.Post{
					ID: 1,
				},
			},
			LastUpdated: 54,
		},
		{
			StandalonePost: common.StandalonePost{
				Board: "a",
				Post: common.Post{
					ID: 2,
				},
			},
			LastUpdated: 55,
		},
	})

	ctr, err := BoardCounter("a")
	if err != nil {
		t.Fatal(err)
	}
	if ctr != 55 {
		LogUnexpected(t, 55, ctr)
	}
}

func TestReservePostID(t *testing.T) {
	assertTableClear(t, "main")
	assertInsert(t, "main", map[string]interface{}{
		"id":      "info",
		"postCtr": 0,
	})

	for i := uint64(1); i <= 2; i++ {
		id, err := ReservePostID()
		if err != nil {
			t.Fatal(err)
		}
		if id != i {
			LogUnexpected(t, i, id)
		}
	}
}
