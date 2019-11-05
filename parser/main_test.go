package parser

import (
	"os"
	"testing"

	"github.com/Chiiruno/meguca/config"
	"github.com/Chiiruno/meguca/db"
)

func TestMain(m *testing.M) {
	err := config.Server.Load()
	if err != nil {
		panic(err)
	}
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
