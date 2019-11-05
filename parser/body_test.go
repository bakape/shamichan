package parser

import (
	"database/sql"
	"testing"
	"time"

	"github.com/Chiiruno/meguca/common"
	"github.com/Chiiruno/meguca/config"
	"github.com/Chiiruno/meguca/db"
	. "github.com/Chiiruno/meguca/test"
	"github.com/Chiiruno/meguca/test/test_db"
)

func TestParseLine(t *testing.T) {
	config.SetBoardConfigs(config.BoardConfigs{
		ID: "a",
	})

	links, com, err := ParseBody([]byte("#flip,"), "a", 1, 1, "::1", false)
	if err != nil {
		t.Fatal(err)
	}
	if links != nil {
		t.Fatalf("unexpected links: %#v", links)
	}
	if com == nil {
		t.Fatalf("no commands")
	}
	if com[0].Type != common.Flip {
		t.Fatalf("unexpected command type: %d", com[0].Type)
	}
}

func TestParseBody(t *testing.T) {
	test_db.ClearTables(t, "boards")
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
	err := db.InTransaction(false, func(tx *sql.Tx) error {
		for _, p := range posts {
			err := db.WritePost(tx, p)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	links, com, err := ParseBody(
		[]byte("#flip?\n>>8\n>>>6 \n(#flip)\n>foo #flip bar \n#flip"),
		"a",
		1,
		1,
		"::1",
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if l := len(com); l != 3 {
		t.Errorf("unexpected command count: %d", l)
	}
	AssertEquals(t, links, []common.Link{
		{8, 1, "a"},
		{6, 1, "a"},
	})
}

func writeSampleBoard(t *testing.T) {
	t.Helper()

	b := db.BoardConfigs{
		BoardConfigs: config.BoardConfigs{
			ID:        "a",
			Eightball: []string{"yes"},
		},
	}
	err := db.InTransaction(false, func(tx *sql.Tx) error {
		return db.WriteBoard(tx, b)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func writeSampleThread(t *testing.T) {
	t.Helper()

	thread := db.Thread{
		ID:    1,
		Board: "a",
	}
	op := db.Post{
		StandalonePost: common.StandalonePost{
			Post: common.Post{
				ID:   1,
				Time: time.Now().Unix(),
			},
			OP: 1,
		},
		IP: "::1",
	}
	err := db.WriteThread(thread, op)
	if err != nil {
		t.Fatal(err)
	}
}
