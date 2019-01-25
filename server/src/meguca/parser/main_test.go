package parser

import (
	"os"
	"testing"

	"meguca/config"
	"meguca/db"
)

func TestMain(m *testing.M) {
	close, err := db.LoadTestDB("parser")
	if err != nil {
		panic(err)
	}

	config.Set(config.Configs{})

	code := m.Run()
	err = close()
	if err != nil {
		panic(err)
	}
	os.Exit(code)
}

func assertTableClear(t *testing.T, tables ...string) {
	t.Helper()
	if err := db.ClearTables(tables...); err != nil {
		t.Fatal(err)
	}
}
