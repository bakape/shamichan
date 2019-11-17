package db

import (
	"testing"

	"github.com/bakape/meguca/common"
	"github.com/jackc/pgx"
)

func writeSampleThread(t *testing.T) (op Post) {
	t.Helper()

	op = Post{
		StandalonePost: common.StandalonePost{
			Post: common.Post{
				Body: []byte("Slut!"),
			},
		},
		IP: "::1",
	}
	err := InTransaction(func(tx *pgx.Tx) error {
		return InsertThread(tx, "test", &op)
	})
	if err != nil {
		t.Fatal(err)
	}

	return
}

func insertPost(t *testing.T, p *Post) {
	t.Helper()

	prepareForPostInsertion(t, p)
	err := InTransaction(func(tx *pgx.Tx) error {
		return InsertPost(tx, p)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func prepareForPostInsertion(t *testing.T, p *Post) {
	t.Helper()

	assertTableClear(t, "accounts")
	op := writeSampleThread(t)
	p.OP = op.ID
}

func TestInsertPost(t *testing.T) {
	p := Post{
		StandalonePost: common.StandalonePost{
			OP: 1,
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

// func TestSageAndTimestampUpdates(t *testing.T) {
// 	prepareForPostInsertion(t)

// 	var bumpTime, updateTime int64

// 	// Read initial values
// 	buf, err := GetThread(1, 0)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	var thread common.Thread
// 	test.DecodeJSON(t, buf, &thread)
// 	bumpTime = thread.BumpTime
// 	updateTime = thread.UpdateTime

// 	cases := [...]struct {
// 		name string
// 		sage bool
// 	}{
// 		{"no sage", false},
// 		{"with sage", true},
// 	}

// 	for i := range cases {
// 		c := cases[i]
// 		t.Run(c.name, func(t *testing.T) {
// 			time.Sleep(2 * time.Second) // Wait for timestamps to update

// 			p := Post{
// 				StandalonePost: common.StandalonePost{
// 					OP: 1,
// 					Post: common.Post{
// 						Sage: c.sage,
// 					},
// 				},
// 				IP:       "::1",
// 				Password: []byte("6+53653cs3ds"),
// 			}
// 			err := InTransaction(func(tx *pgx.Tx) error {
// 				return InsertPost(tx, &p)
// 			})
// 			if err != nil {
// 				t.Fatal(err)
// 			}

// 			buf, err := GetThread(1, 0)
// 			if err != nil {
// 				t.Fatal(err)
// 			}
// 			var thread common.Thread
// 			test.DecodeJSON(t, buf, &thread)

// 			if thread.UpdateTime <= updateTime {
// 				t.Error("update time not increased")
// 			}
// 			updateTime = thread.UpdateTime

// 			if c.sage {
// 				if thread.BumpTime != bumpTime {
// 					t.Error("bump time changed")
// 				}
// 			} else {
// 				if thread.BumpTime <= bumpTime {
// 					t.Error("bump time not increased")
// 				}
// 			}
// 			bumpTime = thread.BumpTime
// 		})
// 	}
// }
