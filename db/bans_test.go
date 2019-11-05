package db

import (
	"testing"
	"time"

	"github.com/Chiiruno/meguca/common"
	. "github.com/Chiiruno/meguca/test"
)

func TestBanUnban(t *testing.T) {
	prepareForModeration(t)

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
