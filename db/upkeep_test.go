package db

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/test"
)

const eightDays = time.Hour * 24 * 8

type threadExpiryCases []struct {
	id    uint64
	board string
	time  time.Time
}

func TestOpenPostClosing(t *testing.T) {
	assertTableClear(t, "boards")
	writeSampleBoard(t)
	writeSampleThread(t)

	tooOld := time.Now().Add(-time.Minute * 31).Unix()
	posts := [...]Post{
		{
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					ID:      2,
					Editing: true,
					Time:    tooOld,
				},
				OP: 1,
			},
		},
		{
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					ID:      3,
					Editing: true,
					Time:    time.Now().Unix(),
				},
				OP: 1,
			},
		},
	}
	err := InTransaction(false, func(tx *sql.Tx) error {
		for _, p := range posts {
			err := WritePost(tx, p)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := closeDanglingPosts(); err != nil {
		t.Fatal(err)
	}

	cases := [...]struct {
		name    string
		id      uint64
		editing bool
	}{
		{"closed", 2, false},
		{"untouched", 3, true},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			var editing bool
			err := db.
				QueryRow(`SELECT editing FROM posts WHERE id = $1`, c.id).
				Scan(&editing)
			if err != nil {
				t.Fatal(err)
			}
			if editing != c.editing {
				test.LogUnexpected(t, c.editing, editing)
			}
		})
	}

}

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

func assertBoardDeleted(t *testing.T, id string, del bool) {
	t.Helper()

	q := fmt.Sprintf(`from boards where id = '%s'`, id)
	assertDeleted(t, q, del)
}

func assertThreadDeleted(t *testing.T, id uint64, del bool) {
	t.Helper()

	q := fmt.Sprintf(`from threads where id = '%d'`, id)
	assertDeleted(t, q, del)
}

func TestDeleteUnusedBoards(t *testing.T) {
	assertTableClear(t, "boards", "accounts")
	config.Set(config.Configs{
		BoardExpiry: 7,
	})
	writeAllBoard(t)

	t.Run("no boards", func(t *testing.T) {
		(*config.Get()).PruneBoards = true

		if err := deleteUnusedBoards(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("board with no threads", testBoardNoThreads)
	t.Run("pruning disabled", testBoardPruningDisabled)
	t.Run("board with threads", testDeleteUnusedBoards)
}

// Board creation time is not passed in standard board creation
func patchBoardCreationTime(t *testing.T, id string, ti time.Time) {
	t.Helper()

	_, err := sq.Update("boards").
		Set("created", ti).
		Where("id = ?", id).
		Exec()
	if err != nil {
		t.Fatal(err)
	}
}

// Restore all board to enable global logging
func writeAllBoard(t *testing.T) {
	t.Helper()

	err := InTransaction(false, func(tx *sql.Tx) (err error) {
		err = WriteBoard(tx, BoardConfigs{
			BoardConfigs: config.AllBoardConfigs.BoardConfigs,
			Created:      time.Now().UTC(),
		})
		if err != nil {
			return
		}
		return CreateSystemAccount(tx)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func testBoardNoThreads(t *testing.T) {
	(*config.Get()).PruneBoards = true

	err := InTransaction(false, func(tx *sql.Tx) error {
		return WriteBoard(tx, BoardConfigs{
			BoardConfigs: config.BoardConfigs{
				ID:        "l",
				Eightball: []string{},
			},
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	patchBoardCreationTime(t, "l", time.Now().Add(-eightDays))

	if err := deleteUnusedBoards(); err != nil {
		t.Fatal(err)
	}
	assertBoardDeleted(t, "l", true)
}

func testBoardPruningDisabled(t *testing.T) {
	(*config.Get()).PruneBoards = false

	err := InTransaction(false, func(tx *sql.Tx) error {
		return WriteBoard(tx, BoardConfigs{
			BoardConfigs: config.BoardConfigs{
				ID:        "x",
				Eightball: []string{},
			},
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	patchBoardCreationTime(t, "x", time.Now().Add(-eightDays))

	if err := deleteUnusedBoards(); err != nil {
		t.Fatal(err)
	}
	assertBoardDeleted(t, "x", false)
}

func testDeleteUnusedBoards(t *testing.T) {
	(*config.Get()).PruneBoards = true
	fresh := time.Now()
	expired := fresh.Add(-eightDays)

	for _, id := range [...]string{"a", "c"} {
		err := InTransaction(false, func(tx *sql.Tx) error {
			return WriteBoard(tx, BoardConfigs{
				BoardConfigs: config.BoardConfigs{
					ID:        id,
					Eightball: []string{},
				},
			})
		})
		if err != nil {
			t.Fatal(err)
		}
		patchBoardCreationTime(t, id, expired)
	}

	writeExpiringThreads(t, threadExpiryCases{
		{1, "a", expired},
		{3, "c", fresh},
	})

	if err := deleteUnusedBoards(); err != nil {
		t.Fatal(err)
	}

	cases := [...]struct {
		name, board string
		deleted     bool
	}{
		{"deleted", "a", true},
		{"deleted", "x", true},
		{"untouched", "c", false},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			assertBoardDeleted(t, c.board, c.deleted)
		})
	}
}

func writeExpiringThreads(t *testing.T, ops threadExpiryCases) {
	t.Helper()

	for _, op := range ops {
		unix := op.time.Unix()
		thread := Thread{
			ID:         op.id,
			Board:      op.board,
			UpdateTime: unix,
			BumpTime:   unix,
		}
		post := Post{
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					ID: op.id,
				},
				Board: op.board,
				OP:    op.id,
			},
		}
		err := InTransaction(false, func(tx *sql.Tx) error {
			return WriteThread(tx, thread, post)
		})
		if err != nil {
			t.Fatal(err)
		}

		// Override bump time from trigger
		_, err = sq.Update("threads").
			SetMap(map[string]interface{}{
				"bump_time": unix,
			}).
			Where("id = ?", op.id).
			Exec()
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestDeleteOldThreads(t *testing.T) {
	assertTableClear(t, "boards")
	writeSampleBoard(t)
	config.Set(config.Configs{
		Public: config.Public{
			ThreadExpiryMin: 7,
			ThreadExpiryMax: 7,
		},
	})

	t.Run("no threads", func(t *testing.T) {
		(*config.Get()).PruneThreads = true
		if err := deleteOldThreads(); err != nil {
			t.Fatal(err)
		}
	})

	writeExpiringThreads(t, threadExpiryCases{
		{1, "a", time.Now().Add(-eightDays)},
		{2, "a", time.Now()},
	})

	t.Run("pruning disabled", func(t *testing.T) {
		(*config.Get()).PruneThreads = false
		if err := deleteOldThreads(); err != nil {
			t.Fatal(err)
		}
		assertThreadDeleted(t, 1, false)
		assertThreadDeleted(t, 2, false)
	})

	t.Run("deleted", func(t *testing.T) {
		(*config.Get()).PruneThreads = true
		if err := deleteOldThreads(); err != nil {
			t.Fatal(err)
		}
		assertThreadDeleted(t, 1, true)
		assertThreadDeleted(t, 2, false)
	})
}

func TestDeleteBoard(t *testing.T) {
	assertTableClear(t, "boards", "accounts")
	writeSampleBoard(t)
	writeAllBoard(t)

	err := DeleteBoard("a", "admin")
	if err != nil {
		t.Fatal(err)
	}
}

func TestRemoveIdentityInfo(t *testing.T) {
	p := Post{
		StandalonePost: common.StandalonePost{
			OP:    1,
			Board: "a",
		},
		IP:       "::1",
		Password: []byte("6+53653cs3ds"),
	}
	insertPost(t, &p)

	_, err := sq.Update("posts").
		Set("time", time.Now().Add(-8*24*time.Hour).Unix()).
		Where("id = ?", p.ID).
		Exec()
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
	err = sq.Select("ip", "password").
		From("posts").
		Where("id = ?", p.ID).
		QueryRow().
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
