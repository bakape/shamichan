package db

import (
	"reflect"
	"testing"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/templates"
)

func TestLoadConfigs(t *testing.T) {
	assertTableClear(t, "main")
	assertInsert(t, "main", ConfigDocument{
		Document{"config"},
		config.Defaults,
	})

	// Intiial configs
	if err := loadConfigs(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(config.Get(), &config.Defaults) {
		logUnexpected(t, &config.Defaults, config.Get())
	}
}

func TestUpdateConfigs(t *testing.T) {
	templates.TemplateRoot = "testdata"
	config.Set(config.Configs{})

	std := config.Configs{}
	std.Boards = []string{"a", "b", "c"}
	if err := updateConfigs(std); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(config.Get(), &std) {
		logUnexpected(t, &std, config.Get())
	}
}
