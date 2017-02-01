package parser

import (
	"testing"
	"time"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	. "github.com/bakape/meguca/test"
)

func TestParseLine(t *testing.T) {
	config.SetBoardConfigs(config.BoardConfigs{
		ID: "a",
	})

	t.Run("commands disabled", func(t *testing.T) {
		links, com, err := ParseBody([]byte("#flip"), "a")
		if err != nil {
			t.Fatal(err)
		}
		if links != nil {
			t.Fatalf("unexpected links: %#v", links)
		}
		AssertDeepEquals(t, com, []common.Command(nil))
	})

	t.Run("commands enabled", func(t *testing.T) {
		config.SetBoardConfigs(config.BoardConfigs{
			ID: "a",
			BoardPublic: config.BoardPublic{
				PostParseConfigs: config.PostParseConfigs{
					HashCommands: true,
				},
			},
		})

		links, com, err := ParseBody([]byte("#flip"), "a")
		if err != nil {
			t.Fatal(err)
		}
		if links != nil {
			t.Fatalf("unexpected links: %#v", links)
		}
		if com[0].Type != common.Flip {
			t.Fatalf("unexpected command type: %d", com[0].Type)
		}
	})
}

func TestParseBody(t *testing.T) {
	assertTableClear(t, "boards")
	writeSampleBoard(t)
	writeSampleThread(t)

	posts := [...]db.Post{
		{
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					ID: 8,
				},
				OP:    1,
				Board: "a",
			},
		},
		{
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					ID: 6,
				},
				OP:    1,
				Board: "a",
			},
		},
	}
	for _, p := range posts {
		if err := db.WritePost(nil, p); err != nil {
			t.Fatal(err)
		}
	}

	config.SetBoardConfigs(config.BoardConfigs{
		ID: "a",
		BoardPublic: config.BoardPublic{
			PostParseConfigs: config.PostParseConfigs{
				HashCommands: true,
			},
		},
	})

	links, com, err := ParseBody([]byte("#flip\n>>8\n>>>6 #flip\n#flip"), "a")
	if err != nil {
		t.Fatal(err)
	}
	if l := len(com); l != 2 {
		t.Errorf("unexpected command count: %d", l)
	}
	AssertDeepEquals(t, links, [][2]uint64{
		{8, 1},
		{6, 1},
	})
}

func writeSampleBoard(t *testing.T) {
	b := db.BoardConfigs{
		BoardConfigs: config.BoardConfigs{
			ID:        "a",
			Eightball: []string{"yes"},
		},
	}
	if err := db.WriteBoard(nil, b); err != nil {
		t.Fatal(err)
	}
}

func writeSampleThread(t *testing.T) {
	thread := db.Thread{
		ID:    1,
		Board: "a",
		Log:   [][]byte{[]byte("123")},
	}
	op := db.Post{
		StandalonePost: common.StandalonePost{
			Post: common.Post{
				ID:   1,
				Time: time.Now().Unix(),
			},
			OP: 1,
		},
	}
	if err := db.WriteThread(nil, thread, op); err != nil {
		t.Fatal(err)
	}
}
