package db

import (
	"database/sql"
	"testing"
	"time"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/test"
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

func insertPost(t *testing.T, p *Post) {
	t.Helper()

	prepareForPostInsertion(t)

	err := InTransaction(false, func(tx *sql.Tx) error {
		return InsertPost(tx, p)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func prepareForPostInsertion(t *testing.T) {
	t.Helper()

	assertTableClear(t, "boards")
	writeSampleBoard(t)
	writeSampleThread(t)

	// Prevent post ID key collision
	_, err := sq.Select("nextval('post_id')").Exec()
	if err != nil {
		t.Fatal(err)
	}
}

func TestInsertPost(t *testing.T) {
	p := Post{
		StandalonePost: common.StandalonePost{
			OP:    1,
			Board: "a",
		},
		IP:       "::1",
		Password: []byte("6+53653cs3ds"),
	}
	insertPost(t, &p)
	if p.Time == 0 {
		t.Fatal(p.Time)
	}
	if p.ID == 0 {
		t.Fatal(p.ID)
	}
}

func TestGetPostPassword(t *testing.T) {
	p := Post{
		StandalonePost: common.StandalonePost{
			OP:    1,
			Board: "a",
		},
		IP:       "::1",
		Password: []byte("6+53653cs3ds"),
	}
	insertPost(t, &p)

	res, err := GetPostPassword(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	test.AssertEquals(t, res, p.Password)
}

func TestSageAndTimestampUpdates(t *testing.T) {
	prepareForPostInsertion(t)

	var bumpTime, updateTime int64

	// Read initial values
	thread, err := GetThread(1, 0)
	if err != nil {
		t.Fatal(err)
	}
	bumpTime = thread.BumpTime
	updateTime = thread.UpdateTime

	cases := [...]struct {
		name string
		sage bool
	}{
		{"no sage", false},
		{"with sage", true},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			time.Sleep(2 * time.Second) // Wait for timestamps to update

			p := Post{
				StandalonePost: common.StandalonePost{
					OP:    1,
					Board: "a",
					Post: common.Post{
						Sage: c.sage,
					},
				},
				IP:       "::1",
				Password: []byte("6+53653cs3ds"),
			}
			err := InTransaction(false, func(tx *sql.Tx) error {
				return InsertPost(tx, &p)
			})
			if err != nil {
				t.Fatal(err)
			}

			thread, err := GetThread(1, 0)
			if err != nil {
				t.Fatal(err)
			}

			if thread.UpdateTime <= updateTime {
				t.Error("update time not increased")
			}
			updateTime = thread.UpdateTime

			if c.sage {
				if thread.BumpTime != bumpTime {
					t.Error("bump time changed")
				}
			} else {
				if thread.BumpTime <= bumpTime {
					t.Error("bump time not increased")
				}
			}
			bumpTime = thread.BumpTime
		})
	}
}
