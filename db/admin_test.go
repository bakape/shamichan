package db

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/test"
	"github.com/jackc/pgx"
)

func prepareForModeration(t *testing.T) {
	t.Helper()
	assertTableClear(t, "accounts", "bans", "mod_log", "boards", "images")

	writeSampleBoard(t)
	writeSampleThread(t)

	writeSampleImage(t)
	insertSampleImage(t)

	writeAllBoard(t)
	writeAdminAccount(t)
}

func writeAdminAccount(t *testing.T) {
	t.Helper()

	err := InTransaction(func(tx *pgx.Tx) (err error) {
		err = RegisterAccount(tx, "admin", samplePasswordHash)
		if err != nil {
			return
		}
		return WriteStaff(tx, "all", map[common.ModerationLevel][]string{
			common.BoardOwner: {"admin"},
		})
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestDeleteImages(t *testing.T) {
	prepareForModeration(t)

	buf, err := GetPost(1)
	if err != nil {
		t.Fatal(err)
	}
	var p common.StandalonePost
	test.DecodeJSON(t, buf, &p)
	if p.Image == nil {
		t.Fatal("no image")
	}

	err = DeleteImages([]uint64{1}, "admin")
	if err != nil {
		t.Fatal(err)
	}

	buf, err = GetPost(1)
	if err != nil {
		t.Fatal(err)
	}
	p = common.StandalonePost{}
	test.DecodeJSON(t, buf, &p)
	if p.Image != nil {
		t.Fatal("image not deleted")
	}
}

func TestSpoilerImages(t *testing.T) {
	prepareForModeration(t)

	buf, err := GetPost(1)
	if err != nil {
		t.Fatal(err)
	}
	var p common.Post
	test.DecodeJSON(t, buf, &p)
	if p.Image.Spoiler {
		t.Fatal("has spoiler")
	}

	err = ModSpoilerImages([]uint64{1}, "admin")
	if err != nil {
		t.Fatal(err)
	}

	buf, err = GetPost(1)
	if err != nil {
		t.Fatal(err)
	}
	test.DecodeJSON(t, buf, &p)
	if !p.Image.Spoiler {
		t.Fatal("no spoiler")
	}
}

func TestDeletePostsByIP(t *testing.T) {
	assertTableClear(t, "accounts", "bans", "mod_log", "boards")

	writeSampleBoard(t)
	writeSampleThread(t)
	writeAllBoard(t)
	writeAdminAccount(t)

	err := InTransaction(func(tx *pgx.Tx) (err error) {
		err = RegisterAccount(tx, "user1", samplePasswordHash)
		if err != nil {
			return
		}
		err = WriteStaff(tx, "a", map[common.ModerationLevel][]string{
			common.BoardOwner: {"user1"},
		})
		if err != nil {
			return
		}

		return RegisterAccount(tx, "user2", samplePasswordHash)
	})
	if err != nil {
		t.Fatal(err)
	}

	err = InTransaction(func(tx *pgx.Tx) (err error) {
		err = WriteBoard(tx, BoardConfigs{
			BoardConfigs: config.BoardConfigs{
				ID:        "b",
				Eightball: []string{},
			},
		})
		if err != nil {
			return
		}
		return WriteThread(
			tx,
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
			},
		)
	})
	if err != nil {
		t.Fatal(err)
	}

	// To prevent ID clash
	err = SetPostCounter(100)
	if err != nil {
		t.Fatal(err)
	}

	var shouldDelete, shouldSkip uint64
	err = InTransaction(func(tx *pgx.Tx) (err error) {
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
		shouldSkip = post.ID

		return
	})
	if err != nil {
		t.Fatal(err)
	}

	err = DeletePostsByIP(1, "user1", time.Hour, "")
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
		err = InTransaction(func(tx *pgx.Tx) (err error) {
			return InsertPost(tx, &post)
		})
		if err != nil {
			t.Fatal(err)
		}
		if !post.IsDeleted() {
			t.Error("deletion not propagateds after insert")
		}
		assertDeleted(t, post.ID, true)
		if len(post.Moderation) == 0 {
			t.Error("no post moderation entries")
		}
	})

	cases := [...]struct {
		name    string
		id      uint64
		deleted bool
	}{
		{"target", 1, true},
		{"same board and ip", shouldDelete, true},
		{"same board, different ip", shouldSkip, false},
		{"different board, same ip", 2, false},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			assertDeleted(t, c.id, c.deleted)
		})
	}

	t.Run("permissions", func(t *testing.T) {
		t.Parallel()

		cases := [...]struct {
			name, account string
			succeed       bool
		}{
			{"has rights", "user1", true},
			{"no rights", "user2", false},
			{"has global rights", "admin", true},
		}

		for i := range cases {
			c := cases[i]
			t.Run(c.account, func(t *testing.T) {
				t.Parallel()

				err := DeletePostsByIP(1, c.account, 0, "")
				if c.succeed {
					if err != nil {
						t.Fatal(err)
					}
				} else if err != common.ErrNoPermissions {
					t.Fatal(err)
				}
			})
		}
	})
}

func TestPurgePost(t *testing.T) {
	prepareForModeration(t)

	// Test initial and repeated purge, when there is no image
	for i := 0; i < 2; i++ {
		var name string
		if i == 0 {
			name = "with image"
		} else {
			name = "without image"
		}
		t.Run(name, func(t *testing.T) {
			err := PurgePost(1, "admin", "test")
			if err != nil {
				t.Fatal(err)
			}

			buf, err := GetPost(1)
			if err != nil {
				t.Fatal(err)
			}
			var post common.Post
			test.DecodeJSON(t, buf, &post)
			test.AssertEquals(t, len(post.Moderation), i+1)
			test.AssertEquals(t, post.Image == nil, true)
			test.AssertEquals(t, post.Body, "")
		})
	}
}

func TestStickyThread(t *testing.T) {
	prepareForModeration(t)

	cases := [...]struct {
		name   string
		sticky bool
	}{
		{"sticky", true},
		{"unsticky", false},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			if err := SetThreadSticky(1, c.sticky); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestLockThread(t *testing.T) {
	prepareForModeration(t)

	cases := [...]struct {
		name string
		lock bool
	}{
		{"lock", true},
		{"unlock", false},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			if err := SetThreadLock(1, c.lock, "admin"); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestStaff(t *testing.T) {
	prepareForModeration(t)

	staff := map[common.ModerationLevel][]string{common.BoardOwner: {"admin"}}
	err := InTransaction(func(tx *pgx.Tx) error {
		return WriteStaff(tx, "a", staff)
	})
	if err != nil {
		t.Fatal(err)
	}

	res, err := GetStaff("a")
	if err != nil {
		t.Fatal(err)
	}
	test.AssertEquals(t, res, staff)
}

func TestGetSameIPPosts(t *testing.T) {
	prepareForModeration(t)
	writeSampleUser(t)
	err := InTransaction(func(tx *pgx.Tx) (err error) {
		return WriteStaff(tx, "a", map[common.ModerationLevel][]string{
			common.BoardOwner: {"admin"},
			common.Janitor:    {sampleUserID},
		})
	})

	buf, err := GetSameIPPosts(1, sampleUserID)
	if err != nil {
		t.Fatal(err)
	}
	var res []common.Post
	err = json.Unmarshal(buf, &res)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(res); n != 1 {
		t.Fatalf("wrong post count: %d", n)
	}
}

func TestGetModLog(t *testing.T) {
	t.Run("ban_unban", TestBanUnban) // So we have something in the log

	_, err := GetModLog("a")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetModLogEntry(t *testing.T) {
	t.Run("ban_unban", TestBanUnban) // So we have something in the log

	var id uint64
	err := sq.Select("id").From("mod_log").Limit(1).QueryRow().Scan(&id)
	if err != nil {
		t.Fatal(err)
	}

	_, err = GetModLogEntry(id)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCanPerform(t *testing.T) {
	prepareForModeration(t)
	writeSampleUser(t)
	err := InTransaction(func(tx *pgx.Tx) error {
		return WriteStaff(tx, "a", map[common.ModerationLevel][]string{
			common.Moderator: []string{sampleUserID},
		})
	})
	if err != nil {
		t.Fatal(err)
	}

	cases := [...]struct {
		name, user, board string
		auth              common.ModerationLevel
		can               bool
	}{
		{"can mod /all/", "admin", "all", common.Admin, true},
		{"can't mod /all/", sampleUserID, "all", common.Admin, false},
		{"admin can mod anything", "admin", "a", common.BoardOwner, true},
		{"can't mod anything", sampleUserID, "all", common.Moderator, false},
		{"can mod own level", sampleUserID, "a", common.Moderator, true},
		{"can mod lower level", sampleUserID, "a", common.Janitor, true},
		{"can't mod higher level", sampleUserID, "a", common.Janitor, true},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			can, err := CanPerform(c.user, c.board, c.auth)
			if err != nil {
				t.Fatal(err)
			}
			test.AssertEquals(t, can, c.can)
		})
	}
}
