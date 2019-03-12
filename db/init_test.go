package db

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	close, err := LoadTestDB("db")
	if err != nil {
		panic(err)
	}
	code := m.Run()
	err = close()
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
