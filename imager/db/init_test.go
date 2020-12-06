package db

import (
	"context"
	"os"
	"testing"

	"github.com/bakape/meguca/imager/config"
	"github.com/jessevdk/go-flags"
)

func TestMain(m *testing.M) {
	code := 1
	err := func() (err error) {
		_, err = flags.
			NewParser(&config.Server, flags.Default|flags.IgnoreUnknown).
			Parse()
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

func clearTables(t *testing.T, tables ...string) {
	t.Helper()
	if err := ClearTables(tables...); err != nil {
		t.Fatal(err)
	}
}

func assertExec(t *testing.T, q string, args ...interface{}) {
	t.Helper()
	_, err := db.Exec(context.Background(), q, args...)
	if err != nil {
		t.Fatal(err)
	}
}
