package db

import (
	"testing"
	"time"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/test"
)

// Only select post updates should bump threads
func TestPostUpdates(t *testing.T) {
	p := Post{
		StandalonePost: common.StandalonePost{
			OP: 1,
			Post: common.Post{
				Editing: true,
			},
		},
		IP: "::1",
	}
	insertPost(t, &p)

	cases := [...]struct {
		name string
		bump bool
		fn   func(t *testing.T)
	}{
		{
			name: "IP deletion",
			fn: func(t *testing.T) {
				_, err := db.Exec(
					`update posts
					set ip = null
					where id = $1`,
					p.ID,
				)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "close post",
			bump: true,
			fn: func(t *testing.T) {
				_, err := db.Exec(
					`update posts
					set editing = false
					where id = $1`,
					p.ID,
				)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		// {
		// 	name: "moderate post",
		// 	bump: true,
		// 	fn: func(t *testing.T) {
		// 		err := DeletePosts([]uint64{p.ID}, "admin")
		// 		if err != nil {
		// 			t.Fatal(err)
		// 		}
		// 	},
		// },
	}

	buf, err := GetThread(1, 0)
	if err != nil {
		t.Fatal(err)
	}

	var lastThread common.Thread
	test.DecodeJSON(t, buf, &lastThread)
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
			test.DecodeJSON(t, buf, &thread)

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

func TestWriteOpenPostBody(t *testing.T) {
	p := Post{
		StandalonePost: common.StandalonePost{
			OP: 1,
			Post: common.Post{
				Editing: true,
			},
		},
		IP: "::1",
	}
	insertPost(t, &p)

	WriteOpenPostBody(p.ID, "old")
	WriteOpenPostBody(p.ID, "new")
	err := FlushOpenPostBodies()
	if err != nil {
		t.Fatal(err)
	}

	var body string
	err = db.
		QueryRow(
			`select body
			from posts
			where id = $1`,
			p.ID,
		).
		Scan(&body)
	if err != nil {
		t.Fatal(err)
	}
	test.AssertEquals(t, body, "new")
}
