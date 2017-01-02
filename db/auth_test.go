package db

import (
	"bytes"
	"testing"

	"github.com/bakape/meguca/auth"
	. "github.com/bakape/meguca/test"
	"github.com/dancannon/gorethink"
)

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
		{"does not exist", "456", gorethink.ErrEmptyResult},
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
