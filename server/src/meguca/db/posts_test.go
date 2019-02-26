package db

import (
	"database/sql"
	"meguca/common"
	"meguca/config"
	"meguca/test"
	"testing"
	"time"
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
	t.Helper()
	b := BoardConfigs{
		BoardConfigs: config.BoardConfigs{
			ID:        "a",
			Eightball: []string{"yes"},
		},
	}
	err := InTransaction(false, func(tx *sql.Tx) error {
		return WriteBoard(tx, b)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func writeSampleThread(t *testing.T) {
	t.Helper()
	thread := Thread{
		ID:    1,
		Board: "a",
	}
	op := Post{
		StandalonePost: common.StandalonePost{
			Post: common.Post{
				ID:   1,
				Time: time.Now().Unix(),
			},
			OP:    1,
			Board: "a",
		},
		IP: "::1",
	}
	if err := WriteThread(thread, op); err != nil {
		t.Fatal(err)
	}
}

func insertPost(t *testing.T) (p Post) {
	t.Helper()

	assertTableClear(t, "boards")
	writeSampleBoard(t)
	writeSampleThread(t)

	// Prevent key collision
	_, err := sq.Select("nextval('post_id')").Exec()
	if err != nil {
		t.Fatal(err)
	}

	p = Post{
		StandalonePost: common.StandalonePost{
			OP:    1,
			Board: "a",
		},
		IP:       "::1",
		Password: []byte("6+53653cs3ds"),
	}
	err = InTransaction(false, func(tx *sql.Tx) error {
		return InsertPost(tx, &p)
	})
	if err != nil {
		t.Fatal(err)
	}
	return
}

func TestInsertPost(t *testing.T) {
	p := insertPost(t)
	if p.Time == 0 {
		t.Fatal(p.Time)
	}
	if p.ID == 0 {
		t.Fatal(p.ID)
	}
}

func TestGetPostPassword(t *testing.T) {
	p := insertPost(t)
	res, err := GetPostPassword(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	test.AssertDeepEquals(t, res, p.Password)
}
