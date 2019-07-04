package parser

import (
	"os"
	"testing"

	"github.com/bakape/meguca/config"
)

func TestMain(m *testing.M) {
	err := config.Server.Load()
	if err != nil {
		panic(err)
	}

	config.Set(config.Configs{})

	os.Exit(m.Run())
}
