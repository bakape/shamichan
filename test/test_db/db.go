// Database testing utility functions

package test_db

import (
	"testing"

	"github.com/bakape/meguca/db"
)

// func WriteSampleThread(t testing.TB) {
// 	t.Helper()

// 	now := time.Now().Unix()
// 	thread := db.Thread{
// 		ID:         1,
// 		Board:      "a",
// 		PostCtr:    0,
// 		ImageCtr:   1,
// 		UpdateTime: now,
// 	}
// 	op := db.Post{
// 		StandalonePost: common.StandalonePost{
// 			Post: common.Post{
// 				ID:   1,
// 				Time: time.Now().Unix(),
// 			},
// 			OP: 1,
// 		},
// 	}
// 	err := db.InTransaction(func(tx *pgx.Tx) error {
// 		return db.WriteThread(tx, thread, op)
// 	})
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// }

func ClearTables(t testing.TB, tables ...string) {
	t.Helper()

	if err := db.ClearTables(tables...); err != nil {
		t.Fatal(err)
	}
}
