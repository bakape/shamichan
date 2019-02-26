package db

import (
	"database/sql"
	"meguca/common"
	"testing"
)

func TestCleanUpOpenPostBodies(t *testing.T) {
	assertTableClear(t, "boards")
	writeSampleBoard(t)
	writeSampleThread(t)
	err := InTransaction(false, func(tx *sql.Tx) (err error) {
		for _, p := range [...]struct {
			open bool
			id   uint64
		}{
			{false, 16},
			{true, 18},
		} {
			err = WritePost(tx, Post{
				StandalonePost: common.StandalonePost{
					OP:    1,
					Board: "a",
					Post: common.Post{
						ID:      p.id,
						Editing: p.open,
					},
				},
			})
			if err != nil {
				return
			}

			err = SetOpenBody(p.id, []byte("foo"))
			if err != nil {
				return
			}
		}
		return
	})
	if err != nil {
		t.Fatal(err)
	}

	err = cleanUpOpenPostBodies()
	if err != nil {
		t.Fatal(err)
	}

	cases := [...]struct {
		name string
		id   uint64
		open bool
	}{
		{"already closed", 16, false},
		{"does not exist", 17, false},
		{"open", 18, true},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			body, err := GetOpenBody(c.id)
			if err != nil {
				t.Fatal(err)
			}
			if c.open {
				if body != "foo" {
					t.Fatal("body deleted")
				}
			} else {
				if body != "" {
					t.Fatal("body not deleted")
				}
			}
		})
	}
}
