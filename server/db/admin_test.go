package db

import (
	"database/sql"
	"meguca/auth"
	. "meguca/test"
	"testing"
)

func prepareForModeration(t *testing.T) {
	t.Helper()
	assertTableClear(t, "accounts", "bans", "mod_log", "boards", "images")

	writeSampleBoard(t)
	writeSampleThread(t)

	writeSampleImage(t)
	insertSampleImage(t)

	err := InTransaction(false, func(tx *sql.Tx) error {
		return CreateAdminAccount(tx)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestModeratePost(t *testing.T) {
	prepareForModeration(t)

	for _, f := range []func(uint64, string) error{
		ModSpoilerImage,
		DeleteImage,
		DeletePost,
	} {
		if err := f(1, "admin"); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPurgePost(t *testing.T) {
	prepareForModeration(t)

	err := PurgePost(1, "admin", "test")
	if err != nil {
		t.Fatal(err)
	}

	post, err := GetPost(1)
	if err != nil {
		t.Fatal(err)
	}
	AssertDeepEquals(t, len(post.Moderation), 1)
	AssertDeepEquals(t, post.Image == nil, true)
	AssertDeepEquals(t, post.Body, "")
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

	staff := map[string][]string{"owners": {"admin"}}
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
	AssertDeepEquals(t, res, staff)
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

func TestGetModLOg(t *testing.T) {
	t.Run("ban_unban", TestBanUnban) // So we have something in the log

	_, err := GetModLog("a")
	if err != nil {
		t.Fatal(err)
	}
}

func TestCanPerform(t *testing.T) {
	prepareForModeration(t)
	writeSampleUser(t)
	err := InTransaction(false, func(tx *sql.Tx) error {
		return WriteStaff(tx, "a", map[string][]string{
			"moderators": []string{sampleUserID},
		})
	})
	if err != nil {
		t.Fatal(err)
	}

	cases := [...]struct {
		name, user, board string
		auth              auth.ModerationLevel
		can               bool
	}{
		{"can mod /all/", "admin", "all", auth.Admin, true},
		{"can't mod /all/", sampleUserID, "all", auth.Admin, false},
		{"admin can mod anything", "admin", "a", auth.BoardOwner, true},
		{"user can't mod anything", sampleUserID, "all", auth.Moderator, false},
		{"can mod own level", sampleUserID, "a", auth.Moderator, true},
		{"can mod lower level", sampleUserID, "a", auth.Janitor, true},
		{"can't mod higher level", sampleUserID, "a", auth.Janitor, true},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			can, err := CanPerform(c.user, c.board, c.auth)
			if err != nil {
				t.Fatal(err)
			}
			AssertDeepEquals(t, can, c.can)
		})
	}
}
