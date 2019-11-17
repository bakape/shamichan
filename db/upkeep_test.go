package db

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/test"
)

const eightDays = time.Hour * 24 * 8

type threadExpiryCases []struct {
	id    uint64
	board string
	time  time.Time
}

// func TestOpenPostClosing(t *testing.T) {
// 	op := writeSampleThread(t)
// 	tooOld := time.Now().Add(-time.Minute * 31).Unix()
// 	posts := [...]Post{
// 		{
// 			StandalonePost: common.StandalonePost{
// 				Post: common.Post{
// 					ID:      2,
// 					Editing: true,
// 					Time:    tooOld,
// 				},
// 				OP: op.ID,
// 			},
// 		},
// 		{
// 			StandalonePost: common.StandalonePost{
// 				Post: common.Post{
// 					ID:      3,
// 					Editing: true,
// 					Time:    time.Now().Unix(),
// 				},
// 				OP: op.ID,
// 			},
// 		},
// 	}
// 	err := InTransaction(func(tx *pgx.Tx) (err error) {
// 		for i := range posts {
// 			err = InsertPost(tx, &posts[i])
// 			if err != nil {
// 				return
// 			}
// 		}
// 		return
// 	})
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// TODO: Patch creation time

// 	if err := closeDanglingPosts(); err != nil {
// 		t.Fatal(err)
// 	}

// 	cases := [...]struct {
// 		name    string
// 		id      uint64
// 		editing bool
// 	}{
// 		{"closed", 2, false},
// 		{"untouched", 3, true},
// 	}

// 	for i := range cases {
// 		c := cases[i]
// 		t.Run(c.name, func(t *testing.T) {
// 			t.Parallel()
// 			var editing bool
// 			err := db.
// 				QueryRow(`SELECT editing FROM posts WHERE id = $1`, c.id).
// 				Scan(&editing)
// 			if err != nil {
// 				t.Fatal(err)
// 			}
// 			if editing != c.editing {
// 				test.LogUnexpected(t, c.editing, editing)
// 			}
// 		})
// 	}

// }

func assertDeleted(t *testing.T, q string, del bool) {
	t.Helper()

	q = fmt.Sprintf(`select exists (select 1 %s)`, q)
	var exists bool
	err := db.QueryRow(q).Scan(&exists)
	if err != nil {
		t.Fatal(err)
	}

	deleted := !exists
	if deleted != del {
		test.LogUnexpected(t, del, deleted)
	}
}

func assertThreadDeleted(t *testing.T, id uint64, del bool) {
	t.Helper()

	q := fmt.Sprintf(`from threads where id = '%d'`, id)
	assertDeleted(t, q, del)
}

// func writeExpiringThreads(t *testing.T, ops threadExpiryCases) {
// 	t.Helper()

// 	for _, op := range ops {
// 		unix := op.time.Unix()
// 		thread := Thread{
// 			ID:         op.id,
// 			UpdateTime: unix,
// 			BumpTime:   unix,
// 		}
// 		post := Post{
// 			StandalonePost: common.StandalonePost{
// 				Post: common.Post{
// 					ID: op.id,
// 				},
// 				OP: op.id,
// 			},
// 		}
// 		err := InTransaction(func(tx *pgx.Tx) error {
// 			return InsertThread(tx, "test", &post)
// 		})
// 		if err != nil {
// 			t.Fatal(err)
// 		}

// 		// Override bump time from trigger
// 		_, err = db.Exec(
// 			`update threads
// 			set bump_time = $1
// 			where id = $2`,
// 			unix,
// 			op,
// 		)
// 		if err != nil {
// 			t.Fatal(err)
// 		}
// 	}
// }

// func TestDeleteOldThreads(t *testing.T) {
// 	assertTableClear(t)
// 	config.Set(config.Configs{
// 		Public: config.Public{
// 			ThreadExpiryMin: 7,
// 			ThreadExpiryMax: 7,
// 		},
// 	})

// 	t.Run("no threads", func(t *testing.T) {
// 		(*config.Get()).PruneThreads = true
// 		if err := deleteOldThreads(); err != nil {
// 			t.Fatal(err)
// 		}
// 	})

// 	writeExpiringThreads(t, threadExpiryCases{
// 		{1, "a", time.Now().Add(-eightDays)},
// 		{2, "a", time.Now()},
// 	})

// 	t.Run("pruning disabled", func(t *testing.T) {
// 		(*config.Get()).PruneThreads = false
// 		if err := deleteOldThreads(); err != nil {
// 			t.Fatal(err)
// 		}
// 		assertThreadDeleted(t, 1, false)
// 		assertThreadDeleted(t, 2, false)
// 	})

// 	t.Run("deleted", func(t *testing.T) {
// 		(*config.Get()).PruneThreads = true
// 		if err := deleteOldThreads(); err != nil {
// 			t.Fatal(err)
// 		}
// 		assertThreadDeleted(t, 1, true)
// 		assertThreadDeleted(t, 2, false)
// 	})
// }

func TestRemoveIdentityInfo(t *testing.T) {
	p := Post{
		StandalonePost: common.StandalonePost{
			OP: 1,
		},
		IP:       "::1",
		Password: []byte("6+53653cs3ds"),
	}
	insertPost(t, &p)

	_, err := db.Exec(
		`update posts
		set time = $1
		where id = $2`,
		time.Now().Add(-8*24*time.Hour).Unix(),
		p.ID,
	)
	if err != nil {
		t.Fatal(err)
	}

	err = removeIdentityInfo()
	if err != nil {
		t.Fatal(err)
	}

	var (
		ip sql.NullString
		pw []byte
	)
	err = db.
		QueryRow(
			`select ip, password
			from posts
			where id = $1`,
			p.ID,
		).
		Scan(&ip, &pw)
	if err != nil {
		t.Fatal(err)
	}
	if ip.String != "" {
		t.Fatal(ip.String)
	}
	if pw != nil {
		t.Fatal(pw)
	}
}
