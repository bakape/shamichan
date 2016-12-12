package parser

import (
	"testing"

	"github.com/bakape/meguca/common"
	. "github.com/bakape/meguca/test"
)

func TestParseLinks(t *testing.T) {
	assertTableClear(t, "posts")
	assertInsert(t, "posts", []common.DatabasePost{
		{
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					ID: 8,
				},
				OP:    2,
				Board: "a",
			},
		},
		{
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					ID: 6,
				},
				OP:    2,
				Board: "a",
			},
		},
	})

	cases := [...]struct {
		name, in string
		links    common.LinkMap
	}{
		{"no links", "foo bar baz", nil},
		{
			"valid links",
			" >>>1  >>6 >>>>8",
			common.LinkMap{
				6: common.Link{
					OP:    2,
					Board: "a",
				},
				8: common.Link{
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

			links, err := parseLinks(c.in)
			if err != nil {
				t.Fatal(err)
			}
			AssertDeepEquals(t, links, c.links)
		})
	}
}
