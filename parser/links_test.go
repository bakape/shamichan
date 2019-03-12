package parser

import (
	"database/sql"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	. "github.com/bakape/meguca/test"
	"github.com/bakape/meguca/test/test_db"
	"testing"
)

func TestParseLinks(t *testing.T) {
	test_db.ClearTables(t, "boards")
	writeSampleBoard(t)
	writeSampleThread(t)
	config.SetBoardConfigs(config.BoardConfigs{
		ID: "a",
	})

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

	cases := [...]struct {
		name, in string
		links    []common.Link
	}{
		{"no links", "foo bar baz", nil},
		{
			"valid links",
			" >>>88  >>6 >>>>8",
			[]common.Link{
				{6, 1, "a"},
				{8, 1, "a"},
			},
		},
		{"all links invalid", " >>88 >>2 >>33", nil},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			links, _, err := ParseBody([]byte(c.in), "a", 1, 1, "::1", false)
			if err != nil {
				t.Fatal(err)
			}
			AssertDeepEquals(t, links, c.links)
		})
	}
}
