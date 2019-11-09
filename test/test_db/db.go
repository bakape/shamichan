// Database testing utility functions

package test_db

import (
	"database/sql"
	"testing"
	"time"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
)

func WriteSampleBoard(t testing.TB) {
	t.Helper()

	b := db.BoardConfigs{
		BoardConfigs: config.BoardConfigs{
			ID:        "a",
			Eightball: []string{"yes"},
		},
	}
	err := db.InTransaction(func(tx *pgx.Tx) error {
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
		ID:         1,
		Board:      "a",
		PostCtr:    0,
		ImageCtr:   1,
		UpdateTime: now,
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
	err := db.InTransaction(func(tx *pgx.Tx) error {
		return db.WriteThread(tx, thread, op)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func ClearTables(t testing.TB, tables ...string) {
	t.Helper()

	if err := db.ClearTables(tables...); err != nil {
		t.Fatal(err)
	}
}
