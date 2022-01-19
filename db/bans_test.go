package db

import (
	"database/sql"
	"testing"
	"time"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	. "github.com/bakape/meguca/test"
)

func TestBanUnban(t *testing.T) {
	prepareForModeration(t)

	err := InTransaction(false, func(tx *sql.Tx) error {
		return Ban(tx, "all", "test", "admin", time.Minute, 1, common.BanPost)
	})
	if err != nil {
		t.Fatal(err)
	}
	err = RefreshBanCache()
	if err != nil {
		t.Fatal(err)
	}

	for _, board := range [...]string{"a", "all"} {
		_, err = IsBanned(board, "::1")
		if err != common.ErrBanned {
			UnexpectedError(t, err)
		}
	}
	err = Unban("a", 1, "admin")
	if err != nil {
		t.Fatal(err)
	}
}

func TestShadowBin(t *testing.T) {
	assertTableClear(t, "accounts", "bans", "mod_log", "boards")

	writeSampleBoard(t)
	writeSampleThread(t)
	writeAllBoard(t)
	writeAdminAccount(t)

	err := InTransaction(false, func(tx *sql.Tx) (err error) {
		err = RegisterAccount(tx, "user1", samplePasswordHash)
		if err != nil {
			return
		}
		return WriteStaff(tx, "a", map[common.ModerationLevel][]string{
			common.BoardOwner: {"user1"},
		})
	})
	if err != nil {
		t.Fatal(err)
	}

	err = InTransaction(false, func(tx *sql.Tx) error {
		return WriteBoard(tx, BoardConfigs{
			BoardConfigs: config.BoardConfigs{
				ID:        "b",
				Eightball: []string{},
			},
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	err = WriteThread(
		Thread{
			ID:    2,
			Board: "b",
		},
		Post{
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					ID:   2,
					Time: time.Now().Unix(),
				},
				OP:    2,
				Board: "b",
			},
			IP: "::1",
		})
	if err != nil {
		t.Fatal(err)
	}

	// To prevent ID clash
	err = SetPostCounter(100)
	if err != nil {
		t.Fatal(err)
	}

	err = InTransaction(false, func(tx *sql.Tx) error {
		return Ban(tx, "a", "", "user1", time.Hour, 1, common.ShadowBinPost)
	})
	if err != nil {
		t.Fatal(err)
	}

	var shouldDelete, shouldSkipIP, shouldSkipBoard uint64
	err = InTransaction(false, func(tx *sql.Tx) (err error) {
		post := Post{
			StandalonePost: common.StandalonePost{
				OP:    1,
				Board: "a",
			},
			IP: "::1",
		}

		err = InsertPost(tx, &post)
		if err != nil {
			return
		}
		shouldDelete = post.ID

		post.ID = 0
		post.IP = "195.77.83.249"
		err = InsertPost(tx, &post)
		if err != nil {
			return
		}
		shouldSkipIP = post.ID

		post.ID = 0
		post.IP = "::1"
		post.Board = "b"
		post.OP = 2
		err = InsertPost(tx, &post)
		if err != nil {
			return
		}
		shouldSkipBoard = post.ID

		return
	})
	if err != nil {
		t.Fatal(err)
	}

	assertDeleted := func(t *testing.T, id uint64, std bool) {
		var deleted bool
		err := db.QueryRow(
			`select exists (select 1
							from post_moderation
							where post_id = $1 and type = $2)`,
			id, common.DeletePost).
			Scan(&deleted)
		if err != nil {
			t.Fatal(err)
		}
		if deleted != std {
			t.Error(deleted)
		}
	}

	// Assert deletion of next created post on same board
	t.Run("delete next insert", func(t *testing.T) {
		post := Post{
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					Time: time.Now().Unix(),
				},
				OP:    1,
				Board: "a",
			},
			IP: "::1",
		}
		err = InTransaction(false, func(tx *sql.Tx) (err error) {
			return InsertPost(tx, &post)
		})
		if err != nil {
			t.Fatal(err)
		}
		assertDeleted(t, post.ID, true)
		if !post.Moderated {
			t.Error("not marked as moderated")
		}
		if len(post.Moderation) == 0 {
			t.Error("no post moderation entries")
		}
	})

	cases := [...]struct {
		name    string
		id      uint64
		deleted bool
	}{
		{"target", 1, false},
		{"same board and ip", shouldDelete, true},
		{"same board, different ip", shouldSkipIP, false},
		{"different board, same ip", shouldSkipBoard, false},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			assertDeleted(t, c.id, c.deleted)
		})
	}

}
