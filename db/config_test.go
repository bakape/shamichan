package db

import (
	"testing"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/templates"
	. "github.com/bakape/meguca/test"
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
	AssertDeepEquals(t, config.Get(), &config.Defaults)
}

func TestUpdateConfigs(t *testing.T) {
	templates.TemplateRoot = "testdata"
	config.Set(config.Configs{})

	std := config.Configs{}
	std.Boards = []string{"a", "b", "c"}
	if err := updateConfigs(std); err != nil {
		t.Fatal(err)
	}
	AssertDeepEquals(t, config.Get(), &std)
}
