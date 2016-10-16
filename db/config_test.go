package db

import (
	"testing"

	"github.com/bakape/meguca/config"
	. "github.com/bakape/meguca/test"
)

func TestLoadConfigs(t *testing.T) {
	assertTableClear(t, "main", "boards")
	assertInsert(t, "main", ConfigDocument{
		Document{"config"},
		config.Defaults,
	})
	assertInsert(t, "boards", []config.BoardConfigs{
		{ID: "a"},
		{ID: "c"},
	})

	// Intiial configs
	if err := loadConfigs(); err != nil {
		t.Fatal(err)
	}

	AssertDeepEquals(t, config.Get(), &config.Defaults)
	AssertDeepEquals(t, config.GetBoards(), []string{"a", "c"})
}

func TestUpdateConfigs(t *testing.T) {
	config.Set(config.Configs{})

	std := config.Configs{}
	std.Hats = true
	if err := updateConfigs(std); err != nil {
		t.Fatal(err)
	}
	AssertDeepEquals(t, config.Get(), &std)
}

func TestUpdateOnRemovedBoard(t *testing.T) {
	config.SetBoards([]string{"a", "x"})
	config.SetBoardConfigs(config.BoardConfigs{
		ID: "a",
	})

	u := boardConfUpdate{
		BoardConfigs: config.BoardConfigs{
			ID: "a",
		},
		Deleted: true,
	}
	if err := updateBoardConfigs(u); err != nil {
		t.Fatal(err)
	}

	AssertDeepEquals(
		t,
		config.GetBoardConfigs("a"),
		config.BoardConfContainer{},
	)
	AssertDeepEquals(t, config.GetBoards(), []string{"x"})
}

func TestUpdateBoatrdConfigs(t *testing.T) {
	config.SetBoardConfigs(config.BoardConfigs{
		ID: "a",
	})

	u := boardConfUpdate{
		BoardConfigs: config.BoardConfigs{
			ID: "a",
			BoardPublic: config.BoardPublic{
				PostParseConfigs: config.PostParseConfigs{
					HashCommands: true,
				},
			},
		},
	}
	if err := updateBoardConfigs(u); err != nil {
		t.Fatal(err)
	}

	AssertDeepEquals(
		t,
		config.GetBoardConfigs("a").BoardConfigs,
		u.BoardConfigs,
	)
}
