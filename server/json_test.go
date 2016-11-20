package server

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bakape/meguca/config"
	. "github.com/bakape/meguca/test"
	"github.com/bakape/meguca/common"
)

var genericImage = &common.Image{
	ImageCommon: common.ImageCommon{
		SHA1: "foo",
	},
}

func removeIndentation(s string) string {
	s = strings.Replace(s, "\t", "", -1)
	s = strings.Replace(s, "\n", "", -1)
	return s
}

func TestServeConfigs(t *testing.T) {
	etag := "foo"
	config.SetClient([]byte{1}, etag)

	rec, req := newPair("/json/config")
	router.ServeHTTP(rec, req)
	assertCode(t, rec, 200)
	assertBody(t, rec, string([]byte{1}))
	assertEtag(t, rec, etag)

	// And with etag
	rec, req = newPair("/json/config")
	req.Header.Set("If-None-Match", etag)
	router.ServeHTTP(rec, req)
	assertCode(t, rec, 304)
}

func TestDetectLastN(t *testing.T) {
	t.Parallel()

	cases := [...]struct {
		name, in string
		out      int
	}{
		{"no query string", "/a/1", 0},
		{"unparsable", "/a/1?last=addsa", 0},
		{"5", "/a/1?last=5", 5},
		{"50", "/a/1?last=50", 50},
		{"invalid number", "/a/1?last=1000", 0},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			req := newRequest(c.in)
			if n := detectLastN(req); n != c.out {
				LogUnexpected(t, c.out, n)
			}
		})
	}
}

func TestPostJSON(t *testing.T) {
	setupPosts(t)
	setBoards(t, "a")

	const postEtag = "qO18VR0TvaL71iNdrFmaIQ"

	cases := [...]struct {
		name, url, header string
		code              int
		etag              string
	}{
		{
			"invalid post number",
			"/post/www",
			"", 400, "",
		},
		{
			"nonexistent post",
			"/post/66",
			"", 404, "",
		},
		{
			"existing post",
			"/post/1",
			"", 200, postEtag,
		},
		{
			"post etag matches",
			"/post/1",
			postEtag, 304, "",
		},
		{
			"invalid thread board",
			"/nope/1",
			"", 404, "",
		},
		{
			"invalid thread number",
			"/a/www",
			"", 404, "",
		},
		{
			"nonexistent thread",
			"/a/22",
			"", 404, "",
		},
		{
			"valid thread",
			"/a/1",
			"", 200, "W/11",
		},
		{
			"thread etags match",
			"/a/1",
			"W/11", 304, "",
		},
		{
			"invalid board",
			"/nope/",
			"", 404, "",
		},
		{
			"valid board",
			"/a/",
			"", 200, "W/7",
		},
		{
			"board etag matches",
			"/a/",
			"W/7", 304, "",
		},
		{
			"all board",
			"/all/",
			"", 200, "W/8",
		},
		{
			"/all/ board etag matches",
			"/all/",
			"W/8", 304, "",
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			rec, req := newPair("/json" + c.url)
			if c.header != "" {
				req.Header.Set("If-None-Match", c.header)
			}
			router.ServeHTTP(rec, req)
			assertCode(t, rec, c.code)
			if c.code == 200 {
				assertEtag(t, rec, c.etag)
			}
		})
	}
}

// Setup the database for testing post-related paths
func setupPosts(t *testing.T) {
	assertTableClear(t, "main", "posts", "threads")
	assertInsert(t, "main", []map[string]interface{}{
		{
			"id":      "info",
			"postCtr": 8,
		},
		{
			"id": "boardCtrs",
			"a":  7,
		},
	})
	assertInsert(t, "threads", common.DatabaseThread{
		ID:    1,
		Board: "a",
	})
	assertInsert(t, "posts", common.DatabasePost{
		StandalonePost: common.StandalonePost{
			Post: common.Post{
				ID: 1,
			},
			Board: "a",
			OP:    1,
		},
		LastUpdated: 11,
	})
}

func TestServeBoardConfigs(t *testing.T) {
	setBoards(t, "a")
	config.AllBoardConfigs.JSON = []byte("foo")
	conf := config.BoardConfigs{
		ID: "a",
		BoardPublic: config.BoardPublic{
			CodeTags: true,
			Title:    "Animu",
			Notice:   "Notice",
			Banners:  []string{},
		},
	}
	config.SetBoardConfigs(conf)

	cases := [...]struct {
		name, url string
		code      int
		body      string
	}{
		{"invalid board", "aaa", 404, ""},
		{"valid board", "a", 200, string(marshalJSON(t, conf.BoardPublic))},
		{"/all/ board", "all", 200, "foo"},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			rec, req := newPair("/json/boardConfig/" + c.url)
			router.ServeHTTP(rec, req)
			assertCode(t, rec, c.code)
			if c.code == 200 {
				assertBody(t, rec, c.body)
			}
		})
	}
}

func TestServeBoardList(t *testing.T) {
	config.ClearBoards()
	conf := [...][2]string{
		{"a", "Animu"},
	}
	for _, c := range conf {
		_, err := config.SetBoardConfigs(config.BoardConfigs{
			ID: c[0],
			BoardPublic: config.BoardPublic{
				Title: c[1],
			},
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	std := removeIndentation(`
[
	{
		"id":"a",
		"title":"Animu"
	}
]`)

	rec, req := newPair("/json/boardList")
	router.ServeHTTP(rec, req)
	assertBody(t, rec, std)
}

func TestServeStaffPosition(t *testing.T) {
	assertTableClear(t, "boards")
	staff := map[string][]string{
		"owners": {"admin"},
	}
	assertInsert(t, "boards", []config.BoardConfigs{
		{
			ID:    "a",
			Staff: staff,
		},
		{
			ID: "b",
		},
		{
			ID:    "c",
			Staff: staff,
		},
	})

	cases := [...]struct {
		name, position, user, res string
	}{
		{"valid query", "owners", "admin", `["a","c"]`},
		{"invalid user", "mod", "admin", "[]"},
		{"invalid position", "owners", "bullshit", "[]"},
		{"both invalid", "bullocks", "bullshit", "[]"},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			path := fmt.Sprintf("/json/positions/%s/%s", c.position, c.user)
			rec, req := newPair(path)
			router.ServeHTTP(rec, req)
			assertCode(t, rec, 200)
			assertBody(t, rec, c.res)
		})
	}
}

func TestServeBoardTimeStamps(t *testing.T) {
	setBoards(t, "a", "c")
	assertTableClear(t, "posts")
	assertInsert(t, "posts", []common.DatabasePost{
		{
			LastUpdated: 1,
			StandalonePost: common.StandalonePost{
				Board: "a",
				Post: common.Post{
					ID: 11,
				},
			},
		},
		{
			LastUpdated: 2,
			StandalonePost: common.StandalonePost{
				Board: "a",
				Post: common.Post{
					ID: 22,
				},
			},
		},
		{
			LastUpdated: 3,
			StandalonePost: common.StandalonePost{
				Board: "c",
				Post: common.Post{
					ID: 33,
				},
			},
		},
	})

	rec, req := newPair("/json/boardTimestamps")
	router.ServeHTTP(rec, req)
	assertBody(t, rec, `{"a":2,"c":3}`)
}
