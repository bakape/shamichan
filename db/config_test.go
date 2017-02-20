package db

import (
	"testing"

	"../config"
	. "../test"
)

func TestLoadConfigs(t *testing.T) {
	config.Clear()
	assertExec(t, `UPDATE main SET val = '{"mature":true}' WHERE id = 'config'`)

	if err := loadConfigs(); err != nil {
		t.Fatal(err)
	}

	AssertDeepEquals(t, config.Get(), &config.Configs{
		Public: config.Public{
			Mature: true,
		},
	})
}

func TestUpdateConfigs(t *testing.T) {
	config.Set(config.Configs{})

	std := config.Configs{
		Public: config.Public{
			Mature: true,
		},
	}
	if err := updateConfigs(`{"mature":true}`); err != nil {
		t.Fatal(err)
	}
	AssertDeepEquals(t, config.Get(), &std)
}

func TestUpdateOnRemovedBoard(t *testing.T) {
	assertTableClear(t, "boards")
	config.Clear()
	config.SetBoardConfigs(config.BoardConfigs{
		ID: "a",
	})

	if err := updateBoardConfigs("a"); err != nil {
		t.Fatal(err)
	}

	AssertDeepEquals(
		t,
		config.GetBoardConfigs("a"),
		config.BoardConfContainer{},
	)
	AssertDeepEquals(t, config.GetBoards(), []string{})
}

func TestUpdateOnAddBoard(t *testing.T) {
	assertTableClear(t, "boards")
	config.Clear()

	std := BoardConfigs{
		BoardConfigs: config.BoardConfigs{
			ID: "a",
			BoardPublic: config.BoardPublic{
				ForcedAnon: true,
			},
			Eightball: []string{"yes"},
		},
	}
	if err := WriteBoard(nil, std); err != nil {
		t.Fatal(err)
	}

	if err := updateBoardConfigs("a"); err != nil {
		t.Fatal(err)
	}

	AssertDeepEquals(
		t,
		config.GetBoardConfigs("a").BoardConfigs,
		std.BoardConfigs,
	)
	AssertDeepEquals(t, config.GetBoards(), []string{"a"})
}

func TestUpdateBoardConfigs(t *testing.T) {
	assertTableClear(t, "boards")
	config.Clear()

	std := BoardConfigs{
		BoardConfigs: config.BoardConfigs{
			ID: "a",
			BoardPublic: config.BoardPublic{
				ForcedAnon: true,
			},
			Eightball: []string{"yes"},
		},
	}
	if err := WriteBoard(nil, std); err != nil {
		t.Fatal(err)
	}

	if err := loadBoardConfigs(); err != nil {
		t.Fatal(err)
	}

	AssertDeepEquals(
		t,
		config.GetBoardConfigs("a").BoardConfigs,
		std.BoardConfigs,
	)

	assertExec(t,
		`UPDATE boards
			SET title = 'foo'
			WHERE id = 'a'`,
	)

	if err := updateBoardConfigs("a"); err != nil {
		t.Fatal(err)
	}

	std.Title = "foo"
	AssertDeepEquals(
		t,
		config.GetBoardConfigs("a").BoardConfigs,
		std.BoardConfigs,
	)
}
