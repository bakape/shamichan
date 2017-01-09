package db

import "testing"

func init() {
	DBName = "meguca_test_db"
	IsTest = true
	if err := LoadDB(); err != nil {
		panic(err)
	}
}

func assertTableClear(t *testing.T, tables ...string) {
	if err := ClearTables(tables...); err != nil {
		t.Fatal(err)
	}
}

func assertExec(t *testing.T, q string, args ...interface{}) {
	_, err := db.Exec(q, args...)
	if err != nil {
		t.Fatal(err)
	}
}
