package parser

import (
	"testing"

	"meguca/config"
	"meguca/db"
)

func init() {
	db.ConnArgs = db.TestConnArgs
	db.IsTest = true
	if err := db.LoadDB(); err != nil {
		panic(err)
	}
	config.Set(config.Configs{})
}

func assertTableClear(t *testing.T, tables ...string) {
	if err := db.ClearTables(tables...); err != nil {
		t.Fatal(err)
	}
}
