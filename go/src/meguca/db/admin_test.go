package db

import (
	. "meguca/test"
	"testing"
	"time"
)

func TestBanUnban(t *testing.T) {
	prepareForModeration(t)

	if err := Ban("a", "test", "admin", time.Now(), 1); err != nil {
		t.Fatal(err)
	}
	if err := Unban("a", 1, "admin"); err != nil {
		t.Fatal(err)
	}
}

func prepareForModeration(t *testing.T) {
	t.Helper()
	assertTableClear(t, "accounts", "bans", "mod_log", "boards")
	writeSampleBoard(t)
	writeSampleThread(t)
	if err := CreateAdminAccount(); err != nil {
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

	err := WriteStaff("a", staff)
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

	res, err := GetSameIPPosts(1, "a")
	if err != nil {
		t.Fatal(err)
	}
	if n := len(res); n != 1 {
		t.Fatalf("wrong post count: %d", n)
	}
}

func TestGetModLOg(t *testing.T) {
	prepareForModeration(t)
	_, err := GetModLog("a")
	if err != nil {
		t.Fatal(err)
	}
}
