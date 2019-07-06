package parser

import (
	"testing"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/test"
)

func TestParseLinks(t *testing.T) {
	t.Parallel()

	cases := [...]struct {
		name, in string
		links    []uint64
	}{
		{"no links", "foo bar baz", nil},
		{
			"valid links",
			" >>6 >>>>8",
			[]uint64{6, 8},
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
