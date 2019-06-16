package db

import (
	"testing"
	"time"

	"github.com/bakape/meguca/common"
)

// Only select post updates should bump threads
func TestNoBumpOnPostUpdate(t *testing.T) {
	p := Post{
		StandalonePost: common.StandalonePost{
			OP:    1,
			Board: "a",
			Post: common.Post{
				Editing: true,
			},
		},
		IP: "::1",
	}
	insertPost(t, &p)

	writeAllBoard(t)
	writeAdminAccount(t)

	cases := [...]struct {
		name string
		bump bool
		fn   func(t *testing.T)
	}{
		{
			name: "IP deletion",
			fn: func(t *testing.T) {
				_, err := sq.
					Update("posts").
					Set("ip", nil).
					Where("id = ?", p.ID).
					Exec()
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "close post",
			bump: true,
			fn: func(t *testing.T) {
				_, err := sq.
					Update("posts").
					Set("editing", false).
					Where("id = ?", p.ID).
					Exec()
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "moderate post",
			bump: true,
			fn: func(t *testing.T) {
				err := DeletePosts([]uint64{p.ID}, "admin")
				if err != nil {
					t.Fatal(err)
				}
			},
		},
	}

	buf, err := GetThread(1, 0)
	if err != nil {
		t.Fatal(err)
	}

	var lastThread common.Thread
	decode(t, buf, &lastThread)
	last := lastThread.BumpTime

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			time.Sleep(time.Second)

			c.fn(t)

			buf, err := GetThread(1, 0)
			if err != nil {
				t.Fatal(err)
			}
			var thread common.Thread
			decode(t, buf, &thread)

			if c.bump {
				if thread.BumpTime == last {
					t.Fatal("bump time not mutated")
				}
				last = thread.BumpTime
			} else {
				if thread.BumpTime != last {
					t.Fatal("bump tim  mutated")
				}
			}
		})
	}
}
