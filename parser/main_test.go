package parser

import (
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"os"
	"testing"
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
