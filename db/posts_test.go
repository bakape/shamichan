package db

import (
	"testing"

	"time"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
)

func TestValidateOp(t *testing.T) {
	assertTableClear(t, "boards")
	writeSampleBoard(t)
	writeSampleThread(t)

	cases := [...]struct {
		id      uint64
		board   string
		isValid bool
	}{
		{1, "a", true},
		{15, "a", false},
	}

	for i := range cases {
		c := cases[i]
		t.Run("", func(t *testing.T) {
			t.Parallel()
			valid, err := ValidateOP(c.id, c.board)
			if err != nil {
				t.Fatal(err)
			}
			if valid != c.isValid {
				t.Fatal("unexpected result")
			}
		})
	}
}

func writeSampleBoard(t *testing.T) {
	b := DatabaseBoardConfigs{
		BoardConfigs: config.BoardConfigs{
			ID:        "a",
			Eightball: []string{"yes"},
		},
	}
	if err := WriteBoard(b, false); err != nil {
		t.Fatal(err)
	}
}

func writeSampleThread(t *testing.T) {
	thread := DatabaseThread{
		ID:    1,
		Board: "a",
		Log:   []string{"123"},
	}
	op := DatabasePost{
		StandalonePost: common.StandalonePost{
			Post: common.Post{
				ID:   1,
				Time: time.Now().Unix(),
			},
			OP: 1,
		},
	}
	if err := WriteThread(thread, op); err != nil {
		t.Fatal(err)
	}
}
