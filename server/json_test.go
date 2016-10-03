package server

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/types"
)

var genericImage = &types.Image{
	ImageCommon: types.ImageCommon{
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
		{"within bounds", "/a/1?last=100", 100},
		{"too large", "/a/1?last=1000", 0},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			req := newRequest(c.in)
			if n := detectLastN(req); n != c.out {
				logUnexpected(t, c.out, n)
			}
		})
	}
}

func TestPostJSON(t *testing.T) {
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
	assertInsert(t, "threads", types.DatabaseThread{
		ID:      1,
		Board:   "a",
		PostCtr: 11,
	})
	assertInsert(t, "posts", types.DatabasePost{
		Post: types.Post{
			ID:    1,
			Board: "a",
			OP:    1,
		},
	})

	(*config.Get()).Boards = []string{"a"}

	const postEtag = "pF2WuTWab2p8BN88aBNTxw"

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
			"nonexitant post",
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
			"", 400, "",
		},
		{
			"nonexitant thread",
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

func TestServeBoardConfigs(t *testing.T) {
	assertTableClear(t, "boards")

	(*config.Get()).Boards = []string{"a"}
	config.AllBoardConfigs = []byte("foo")
	conf := config.BoardConfigs{
		ID:       "a",
		CodeTags: true,
		Title:    "Animu",
		Notice:   "Notice",
		Banners:  []string{},
	}
	assertInsert(t, "boards", conf)

	clientConf, err := conf.MarshalPublicJSON()
	if err != nil {
		t.Fatal(err)
	}

	cases := [...]struct {
		name, url string
		code      int
		body      string
	}{
		{"invalid board", "aaa", 404, ""},
		{"valid board", "a", 200, string(clientConf)},
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
	assertTableClear(t, "boards")

	// No boards
	rec, req := newPair("/json/boardList")
	router.ServeHTTP(rec, req)
	assertBody(t, rec, "[]")

	assertInsert(t, "boards", []config.BoardConfigs{
		{
			ID:    "a",
			Title: "Animu",
		},
		{
			ID:    "g",
			Title: "Technology",
		},
	})

	std := removeIndentation(`
[
	{
		"id":"a",
		"title":"Animu"
	},
	{
		"id":"g",
		"title":"Technology"
	}
]`)

	rec, req = newPair("/json/boardList")
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

// func (d *DB) TestSpoilerImageWrongPassword(c *C) {
// 	const password = "123"
// 	hash, err := auth.BcryptHash(password, 6)
// 	c.Assert(err, IsNil)

// 	thread := types.DatabaseThread{
// 		ID: 1,
// 		Posts: map[int64]types.DatabasePost{
// 			1: {
// 				Password: hash,
// 				Post: types.Post{
// 					ID: 1,
// 					Image: &types.Image{
// 						ImageCommon: types.ImageCommon{
// 							SHA1: "123",
// 						},
// 					},
// 				},
// 			},
// 			2: {
// 				Password: hash,
// 				Post: types.Post{
// 					ID: 1,
// 				},
// 			},
// 		},
// 	}
// 	c.Assert(db.Write(r.Table("threads").Insert(thread)), IsNil)

// 	samples := [...]struct {
// 		id        int64
// 		password  string
// 		code      int
// 		spoilered bool
// 	}{
// 		{2, password, 400, false}, // No image
// 		{1, "122", 403, false},    // Wrong password
// 		{1, password, 200, true},  // Success
// 		{1, password, 200, true},  // Already spoilered
// 	}

// 	for _, s := range samples {
// 		data := spoilerRequest{
// 			ID:       s.id,
// 			Password: s.password,
// 		}
// 		rec, req := newJSONPair(c, "/json/spoiler", data)
// 		d.r.ServeHTTP(rec, req)

// 		assertCode(rec, s.code, c)

// 		var spoilered bool
// 		msg := []byte("11" + strconv.Itoa(int(s.id)))
// 		q := r.And(
// 			db.FindParentThread(s.id).Field("log").Contains(msg),
// 			db.FindPost(s.id).Field("image").Field("spoiler"),
// 		)
// 		c.Assert(db.One(q, &spoilered), IsNil)
// 		c.Assert(spoilered, Equals, s.spoilered)
// 	}
// }
