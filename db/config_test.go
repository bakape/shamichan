package db

import (
	"testing"

	"github.com/bakape/meguca/config"
	. "github.com/bakape/meguca/test"
)

func TestLoadConfigs(t *testing.T) {
	config.Clear()
	assertTableClear(t, "main")
	assertExec(t,
		`INSERT INTO main (id, val) VALUES ('config', $1)`,
		`{"mature":true}`,
	)

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

	std := config.DatabaseBoardConfigs{
		BoardConfigs: config.BoardConfigs{
			ID: "a",
			BoardPublic: config.BoardPublic{
				CodeTags: true,
			},
			Eightball: []string{"yes"},
		},
	}
	if err := WriteBoard(std, false); err != nil {
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

	std := config.DatabaseBoardConfigs{
		BoardConfigs: config.BoardConfigs{
			ID: "a",
			BoardPublic: config.BoardPublic{
				PostParseConfigs: config.PostParseConfigs{
					HashCommands: true,
				},
			},
			Eightball: []string{"yes"},
		},
	}
	if err := WriteBoard(std, false); err != nil {
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
