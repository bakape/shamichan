package db

import (
	"testing"

	. "meguca/test"
)

func TestRegisterAccount(t *testing.T) {
	assertTableClear(t, "accounts")

	const id = "123"
	hash := []byte{1, 2, 3}

	// New user
	if err := RegisterAccount(id, hash); err != nil {
		t.Fatal(err)
	}

	// User name already registered
	if err := RegisterAccount(id, hash); err != ErrUserNameTaken {
		UnexpectedError(t, err)
	}
}
