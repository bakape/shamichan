// Database testing utility functions

package test_db

import (
	"database/sql"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"testing"
	"time"
)

func WriteSampleBoard(t testing.TB) {
	t.Helper()

	b := db.BoardConfigs{
		BoardConfigs: config.BoardConfigs{
			ID:        "a",
			Eightball: []string{"yes"},
		},
	}
	err := db.InTransaction(false, func(tx *sql.Tx) error {
		return db.WriteBoard(tx, b)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func WriteSampleThread(t testing.TB) {
	t.Helper()

	now := time.Now().Unix()
	thread := db.Thread{
		ID:        1,
		Board:     "a",
		PostCtr:   0,
		ImageCtr:  1,
		ReplyTime: now,
	}
	op := db.Post{
		StandalonePost: common.StandalonePost{
			Post: common.Post{
				ID:   1,
				Time: time.Now().Unix(),
			},
			OP: 1,
		},
	}
	if err := db.WriteThread(thread, op); err != nil {
		t.Fatal(err)
	}
}

func ClearTables(t testing.TB, tables ...string) {
	t.Helper()

	if err := db.ClearTables(tables...); err != nil {
		t.Fatal(err)
	}
}
