package db

import (
	"testing"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/test"
)

func TestLoadConfigs(t *testing.T) {
	config.Clear()
	std := config.Configs{
		Public: config.Public{
			Mature: true,
		},
	}
	err := WriteConfigs(std)
	if err != nil {
		t.Fatal(err)
	}

	if err := loadConfigs(); err != nil {
		t.Fatal(err)
	}

	test.AssertEquals(t, config.Get(), &std)
}
