// Database testing utility functions

package test_db

import (
	"context"
	"testing"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/db"
)

// Insert sample thread and return its ID
func InsertSampleThread(t *testing.T) (id uint64, authKey auth.AuthKey) {
	t.Helper()

	authKey = genToken(t)
	id, err := db.InsertThread(context.Background(), db.ThreadInsertParams{
		Subject: "test",
		Tags:    []string{"animu", "mango"},
		PostInsertParamsCommon: db.PostInsertParamsCommon{
			AuthKey: &authKey,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if id == 0 {
		t.Fatal("id not set")
	}
	return
}

func ClearTables(t testing.TB, tables ...string) {
	t.Helper()

	if err := db.ClearTables(tables...); err != nil {
		t.Fatal(err)
	}
}

// Generate random auth.AuthKey
func genToken(t *testing.T) auth.AuthKey {
	t.Helper()

	b, err := auth.NewAuthKey()
	if err != nil {
		t.Fatal(err)
	}
	return b
}
