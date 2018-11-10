package db

import (
	"meguca/common"
	. "meguca/test"
	"testing"
	"time"
)

func TestBanUnban(t *testing.T) {
	prepareForModeration(t)
	writeAllBoard(t)

	err := Ban("all", "test", "admin", time.Minute, 1)
	if err != nil {
		t.Fatal(err)
	}
	err = RefreshBanCache()
	if err != nil {
		t.Fatal(err)
	}

	for _, board := range [...]string{"a", "all"} {
		err = IsBanned(board, "::1")
		if err != common.ErrBanned {
			UnexpectedError(t, err)
		}
	}
	err = Unban("a", 1, "admin")
	if err != nil {
		t.Fatal(err)
	}
}
