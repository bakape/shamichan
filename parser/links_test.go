package parser

import (
	"testing"

	. "github.com/bakape/meguca/test"
	"github.com/bakape/meguca/types"
)

func TestParseLinks(t *testing.T) {
	assertTableClear(t, "posts")
	assertInsert(t, "posts", []types.DatabasePost{
		{
			StandalonePost: types.StandalonePost{
				Post: types.Post{
					ID: 8,
				},
				OP:    2,
				Board: "a",
			},
		},
		{
			StandalonePost: types.StandalonePost{
				Post: types.Post{
					ID: 6,
				},
				OP:    2,
				Board: "a",
			},
		},
	})

	cases := [...]struct {
		name, in string
		links    types.LinkMap
	}{
		{"no links", "foo bar baz", nil},
		{
			"valid links",
			" >>>1  >>6 >>>>8",
			types.LinkMap{
				6: types.Link{
					OP:    2,
					Board: "a",
				},
				8: types.Link{
					OP:    2,
					Board: "a",
				},
			},
		},
		{"all links invalid", " >>1 >>2 >>33", nil},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			links, err := parseLinks([]byte(c.in))
			if err != nil {
				t.Fatal(err)
			}
			AssertDeepEquals(t, links, c.links)
		})
	}
}
