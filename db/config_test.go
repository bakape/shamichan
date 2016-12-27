package db

// import (
// 	"testing"

// 	"github.com/bakape/meguca/config"
// 	. "github.com/bakape/meguca/test"
// )

// func TestLoadConfigs(t *testing.T) {
// 	config.Clear()
// 	assertTableClear(t, "main", "boards")
// 	assertInsert(t, "main", ConfigDocument{
// 		Document{"config"},
// 		config.Defaults,
// 	})

// 	if err := loadConfigs(); err != nil {
// 		t.Fatal(err)
// 	}

// 	AssertDeepEquals(t, config.Get(), &config.Defaults)
// }

// func TestUpdateConfigs(t *testing.T) {
// 	config.Set(config.Configs{})

// 	std := config.Configs{}
// 	std.Mature = true
// 	if err := updateConfigs(std); err != nil {
// 		t.Fatal(err)
// 	}
// 	AssertDeepEquals(t, config.Get(), &std)
// }

// func TestUpdateOnRemovedBoard(t *testing.T) {
// 	config.Clear()
// 	config.SetBoardConfigs(config.BoardConfigs{
// 		ID: "a",
// 	})

// 	u := boardConfUpdate{
// 		BoardConfigs: config.BoardConfigs{
// 			ID: "a",
// 		},
// 		Deleted: true,
// 	}
// 	if err := updateBoardConfigs(u); err != nil {
// 		t.Fatal(err)
// 	}

// 	AssertDeepEquals(
// 		t,
// 		config.GetBoardConfigs("a"),
// 		config.BoardConfContainer{},
// 	)
// 	AssertDeepEquals(t, config.GetBoards(), []string{})
// }

// func TestUpdateOnAddBoard(t *testing.T) {
// 	config.Clear()

// 	std := config.BoardConfigs{
// 		ID: "a",
// 		BoardPublic: config.BoardPublic{
// 			CodeTags: true,
// 		},
// 	}
// 	u := boardConfUpdate{
// 		BoardConfigs: std,
// 	}

// 	if err := updateBoardConfigs(u); err != nil {
// 		t.Fatal(err)
// 	}

// 	AssertDeepEquals(t, config.GetBoardConfigs("a").BoardConfigs, std)
// 	AssertDeepEquals(t, config.GetBoards(), []string{"a"})
// }

// func TestUpdateBoardConfigs(t *testing.T) {
// 	config.SetBoardConfigs(config.BoardConfigs{
// 		ID: "a",
// 	})

// 	u := boardConfUpdate{
// 		BoardConfigs: config.BoardConfigs{
// 			ID: "a",
// 			BoardPublic: config.BoardPublic{
// 				PostParseConfigs: config.PostParseConfigs{
// 					HashCommands: true,
// 				},
// 			},
// 		},
// 	}
// 	if err := updateBoardConfigs(u); err != nil {
// 		t.Fatal(err)
// 	}

// 	AssertDeepEquals(
// 		t,
// 		config.GetBoardConfigs("a").BoardConfigs,
// 		u.BoardConfigs,
// 	)
// }
