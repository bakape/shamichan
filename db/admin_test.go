package db

import (
	"database/sql"
	"testing"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/test"
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

	err := InTransaction(false, func(tx *sql.Tx) (err error) {
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

	p, err := GetPost(1)
	if err != nil {
		t.Fatal(err)
	}
	if p.Image == nil {
		t.Fatal("no image")
	}

	err = InTransaction(false, func(tx *sql.Tx) error {
		return DeleteImages(tx, 1, "admin", false)
	})
	if err != nil {
		t.Fatal(err)
	}

	p, err = GetPost(1)
	if err != nil {
		t.Fatal(err)
	}
	if p.Image != nil {
		t.Fatal("image not deleted")
	}
}

func TestSpoilerImages(t *testing.T) {
	prepareForModeration(t)

	p, err := GetPost(1)
	if err != nil {
		t.Fatal(err)
	}
	if p.Image.Spoiler {
		t.Fatal("has spoiler")
	}

	err = InTransaction(false, func(tx *sql.Tx) error {
		return ModSpoilerImages(tx, 1, "admin", false)
	})
	if err != nil {
		t.Fatal(err)
	}

	p, err = GetPost(1)
	if err != nil {
		t.Fatal(err)
	}
	if !p.Image.Spoiler {
		t.Fatal("no spoiler")
	}
}

func TestPurgePost(t *testing.T) {
	prepareForModeration(t)

	err := InTransaction(false, func(tx *sql.Tx) error {
		return PurgePost(tx, 1, "admin", "test", false)
	})
	if err != nil {
		t.Fatal(err)
	}

	post, err := GetPost(1)
	if err != nil {
		t.Fatal(err)
	}
	test.AssertEquals(t, len(post.Moderation), 1)
	test.AssertEquals(t, post.Image == nil, true)
	test.AssertEquals(t, post.Body, "")
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
	err := InTransaction(false, func(tx *sql.Tx) error {
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

	res, err := GetSameIPPosts(1, "a", sampleUserID)
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
	err := InTransaction(false, func(tx *sql.Tx) error {
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
		{"can't mod higher level", sampleUserID, "a", common.BoardOwner, false},
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
