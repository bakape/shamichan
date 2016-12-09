package db

import (
	"testing"

	"bytes"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	. "github.com/bakape/meguca/test"
	r "github.com/dancannon/gorethink"
)

func TestValidateOp(t *testing.T) {
	assertTableClear(t, "threads")
	assertInsert(t, "threads", common.DatabaseThread{
		ID:    1,
		Board: "a",
	})

	samples := [...]struct {
		id      uint64
		board   string
		isValid bool
	}{
		{1, "a", true},
		{15, "a", false},
	}

	for i := range samples {
		s := samples[i]
		t.Run("", func(t *testing.T) {
			t.Parallel()
			valid, err := ValidateOP(s.id, s.board)
			if err != nil {
				t.Fatal(err)
			}
			if valid != s.isValid {
				t.Fatal("unexpected result")
			}
		})
	}
}

func TestThreadCounter(t *testing.T) {
	assertTableClear(t, "posts")
	assertInsert(t, "posts", []common.DatabasePost{
		{
			StandalonePost: common.StandalonePost{
				OP: 1,
				Post: common.Post{
					ID: 1,
				},
			},
			LastUpdated: 54,
		},
		{
			StandalonePost: common.StandalonePost{
				OP: 1,
				Post: common.Post{
					ID: 2,
				},
			},
			LastUpdated: 55,
		},
	})

	ctr, err := ThreadCounter(1)
	if err != nil {
		t.Fatal(err)
	}
	if ctr != 55 {
		LogUnexpected(t, 55, ctr)
	}
}

func TestBoardCounter(t *testing.T) {
	assertTableClear(t, "posts")
	assertInsert(t, "posts", []common.DatabasePost{
		{
			StandalonePost: common.StandalonePost{
				Board: "a",
				Post: common.Post{
					ID: 1,
				},
			},
			LastUpdated: 54,
		},
		{
			StandalonePost: common.StandalonePost{
				Board: "a",
				Post: common.Post{
					ID: 2,
				},
			},
			LastUpdated: 55,
		},
	})

	ctr, err := BoardCounter("a")
	if err != nil {
		t.Fatal(err)
	}
	if ctr != 55 {
		LogUnexpected(t, 55, ctr)
	}
}

func TestRegisterAccount(t *testing.T) {
	assertTableClear(t, "accounts")

	const id = "123"
	hash := []byte{1, 2, 3}
	user := auth.User{
		ID:       id,
		Password: hash,
		Sessions: []auth.Session{},
	}

	// New user
	if err := RegisterAccount(id, hash); err != nil {
		t.Fatal(err)
	}
	var res auth.User
	if err := One(GetAccount(id), &res); err != nil {
		t.Error(err)
	}
	AssertDeepEquals(t, res, user)

	// User name already registered
	if err := RegisterAccount(id, hash); err != ErrUserNameTaken {
		UnexpectedError(t, err)
	}
}

func TestGetLoginHash(t *testing.T) {
	assertTableClear(t, "accounts")

	const id = "123"
	hash := []byte{1, 2, 3}
	assertInsert(t, "accounts", auth.User{
		ID:       id,
		Password: hash,
	})

	samples := [...]struct {
		name, id string
		err      error
	}{
		{"exists", id, nil},
		{"does not exist", "456", r.ErrEmptyResult},
	}

	for i := range samples {
		s := samples[i]
		t.Run(s.name, func(t *testing.T) {
			t.Parallel()
			h, err := GetLoginHash(s.id)
			if err != s.err {
				LogUnexpected(t, s.err, err)
			}
			if s.err == nil {
				if !bytes.Equal(h, hash) {
					LogUnexpected(t, hash, h)
				}
			}
		})
	}
}

func TestReservePostID(t *testing.T) {
	assertTableClear(t, "main")
	assertInsert(t, "main", map[string]interface{}{
		"id":      "info",
		"postCtr": 0,
	})

	for i := uint64(1); i <= 2; i++ {
		id, err := ReservePostID()
		if err != nil {
			t.Fatal(err)
		}
		if id != i {
			LogUnexpected(t, i, id)
		}
	}
}
