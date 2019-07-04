package parser

import (
	"testing"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/test"
)

func TestParseLinks(t *testing.T) {
	t.Parallel()

	cases := [...]struct {
		name, in string
		links    map[uint64]common.Link
	}{
		{"no links", "foo bar baz", nil},
		{
			"valid links",
			" >>6 >>>>8",
			map[uint64]common.Link{
				6: {
					OP:    1,
					Board: "a",
				},
				8: {
					OP:    1,
					Board: "a",
				},
			},
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			links, _, err := ParseBody(
				[]byte(c.in),
				config.BoardConfigs{
					ID: "a",
				},
				false,
			)
			if err != nil {
				t.Fatal(err)
			}
			test.AssertEquals(t, links, c.links)
		})
	}
}
