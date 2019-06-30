package server

import (
	"strings"
	"testing"

	"github.com/bakape/meguca/cache"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/test/test_db"
)

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

func TestPostJSON(t *testing.T) {
	setupPosts(t)
	setBoards(t, "a")
	cache.Clear()

	const postEtag = "PaeloSx234o+Muw3I8u/2g"

	cases := [...]struct {
		name, url, header string
		code              int
		etag              string
	}{
		{
			name: "invalid post number",
			url:  "/post/www",
			code: 400,
		},
		{
			name: "nonexistent post",
			url:  "/post/66",
			code: 404,
		},
		{
			name: "existing post",
			url:  "/post/1",
			code: 200,
		},
		{
			name: "invalid thread number",
			url:  "/boards/a/www",
			code: 404,
		},
		{
			name: "nonexistent thread",
			url:  "/boards/a/22",
			code: 404,
		},
		{
			name: "invalid board",
			url:  "/boards/nope/",
			code: 404,
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
			if c.code == 200 && c.etag != "" {
				assertEtag(t, rec, c.etag)
			}
		})
	}
}

// Setup the database for testing post-related paths
func setupPosts(t *testing.T) {
	t.Helper()

	test_db.ClearTables(t, "boards")
	if err := db.SetPostCounter(11); err != nil {
		t.Fatal(err)
	}
	writeSampleBoard(t)
	writeSampleThread(t)
}

func TestServeBoardConfigs(t *testing.T) {
	setBoards(t, "a")
	config.AllBoardConfigs.JSON = []byte("foo")
	conf := config.BoardConfigs{
		ID: "a",
		BoardPublic: config.BoardPublic{
			Title:  "Animu",
			Notice: "Notice",
		},
	}
	config.SetBoardConfigs(conf)
	config.SetBoardConfigs(config.AllBoardConfigs.BoardConfigs)

	cases := [...]struct {
		name, url string
		code      int
		body      string
	}{
		{"invalid board", "aaa", 404, ""},
		{"valid board", "a", 200, string(marshalJSON(t, conf.BoardPublic))},
		{"/all/ board", "all", 200, string(marshalJSON(t, config.
			AllBoardConfigs.BoardConfigs.BoardPublic))},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			rec, req := newPair("/json/board-config/" + c.url)
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
	},
	{
		"id":"all",
		"title":"Aggregator metaboard"
	}
]`)

	rec, req := newPair("/json/board-list")
	router.ServeHTTP(rec, req)
	assertBody(t, rec, std)
}

func TestServeExtensionMap(t *testing.T) {
	t.Parallel()
	rec, req := newPair("/json/extensions")
	router.ServeHTTP(rec, req)
	assertCode(t, rec, 200)
}

func TestThreadJSON(t *testing.T) {
	cache.Clear()
	test_db.ClearTables(t, "boards")
	writeSampleBoard(t)
	writeSampleThread(t)
	setBoards(t, "a")

	cases := [...]struct {
		name, url string
		code      int
	}{
		{"unparsable thread number", "/www", 404},
		{"valid thread", "/1", 200},
		{"invalid page", "/1?page=-2", 404},
		{"valid page", "/1?page=0", 200},
		{"last page", "/1?page=-1", 200},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			rec, req := newPair("/json/thread" + c.url)
			router.ServeHTTP(rec, req)
			assertCode(t, rec, c.code)
		})
	}
}
