package db

import (
	"os"
	"testing"

	"github.com/bakape/meguca/config"
)

func TestMain(m *testing.M) {
	code := 1
	err := func() (err error) {
		err = config.Server.Load()
		if err != nil {
			return
		}
		err = LoadTestDB()
		if err != nil {
			return
		}
		code = m.Run()
		return
	}()
	if err != nil {
		panic(err)
	}
	os.Exit(code)
}

func assertTableClear(t *testing.T, tables ...string) {
	t.Helper()
	if err := ClearTables(tables...); err != nil {
		t.Fatal(err)
	}
}

func assertExec(t *testing.T, q string, args ...interface{}) {
	t.Helper()
	_, err := db.Exec(q, args...)
	if err != nil {
		t.Fatal(err)
	}
}
