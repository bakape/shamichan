package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/types"
	"github.com/bakape/meguca/util"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

// Does not seem like we can easily resuse testing functions. Thus copy/paste
// for now.
type DB struct {
	r http.Handler
}

var _ = Suite(&DB{})

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

func (d *DB) SetUpSuite(c *C) {
	db.DBName = db.UniqueDBName()
	c.Assert(db.Connect(), IsNil)
	c.Assert(db.InitDB(), IsNil)
	config.Set(config.Configs{})
	d.r = createRouter()
}

func (*DB) SetUpTest(c *C) {
	enableGzip = false
	c.Assert(db.ClearTables(), IsNil)
	config.Set(config.Configs{
		Boards: []string{"a"},
	})
}

func (d *DB) TearDownSuite(c *C) {
	c.Assert(r.DBDrop(db.DBName).Exec(db.RSession), IsNil)
	c.Assert(db.RSession.Close(), IsNil)
}

// Create a multipurpose set of threads and posts for tests
func setupPosts(c *C) {
	threads := []types.DatabaseThread{
		{
			ID:    1,
			Board: "a",
			Log:   dummyLog(11),
			Posts: map[int64]types.DatabasePost{
				1: {
					Post: types.Post{
						ID:    1,
						Image: genericImage,
					},
				},
				2: {
					Post: types.Post{
						ID: 2,
					},
				},
			},
		},
		{
			ID:    3,
			Board: "a",
			Log:   dummyLog(33),
			Posts: map[int64]types.DatabasePost{
				3: {
					Post: types.Post{
						ID:    3,
						Image: genericImage,
					},
				},
			},
		},
		{
			ID:    4,
			Board: "c",
			Log:   dummyLog(44),
			Posts: map[int64]types.DatabasePost{
				4: {
					Post: types.Post{
						ID:    4,
						Image: genericImage,
					},
				},
			},
		},
	}
	c.Assert(db.Write(r.Table("threads").Insert(threads)), IsNil)

	mains := []map[string]interface{}{
		{
			"id":      "info",
			"postCtr": 8,
		},
		{
			"id": "boardCtrs",
			"a":  7,
		},
	}
	c.Assert(db.Write(r.Table("main").Insert(mains)), IsNil)
}

func (w *WebServer) TestConfigServing(c *C) {
	etag := "foo"
	config.SetClient([]byte{1}, etag)

	rec, req := newPair(c, "/json/config")
	w.r.ServeHTTP(rec, req)
	assertCode(rec, 200, c)
	assertBody(rec, string([]byte{1}), c)
	assertEtag(rec, etag, c)

	// And with etag
	rec, req = newPair(c, "/json/config")
	req.Header.Set("If-None-Match", etag)
	w.r.ServeHTTP(rec, req)
	assertCode(rec, 304, c)
}

func (*WebServer) TestDetectLastN(c *C) {
	// No ?last query string
	req := newRequest(c, "/a/1")
	c.Assert(detectLastN(req), Equals, 0)

	// ?last value within bounds
	req = newRequest(c, "/a/1?last=100")
	c.Assert(detectLastN(req), Equals, 100)

	// ?lastNvalue beyond max
	req = newRequest(c, "/a/1?last=1000")
	c.Assert(detectLastN(req), Equals, 0)
}

func (d *DB) TestServePost(c *C) {
	setupPosts(c)

	// Invalid post number
	rec, req := newPair(c, "/json/post/www")
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 400, c)

	// Non-existing post or otherwise invalid post
	rec, req = newPair(c, "/json/post/66")
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 404, c)

	// Existing post
	const (
		body = `{"editing":false,"id":2,"time":0,"body":"","op":1,"board":"a"}`
	)
	etag := util.HashBuffer([]byte(body))
	rec, req = newPair(c, "/json/post/2")
	d.r.ServeHTTP(rec, req)
	assertBody(rec, body, c)
	assertEtag(rec, etag, c)

	// Etags match
	rec, req = newPair(c, "/json/post/2")
	req.Header.Set("If-None-Match", etag)
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 304, c)
}

func (d *DB) TestBoardJSON(c *C) {
	setupPosts(c)

	// Invalid board
	rec, req := newPair(c, "/json/nope/")
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 404, c)

	rec, req = newPair(c, "/json/a/")
	const body = `
{
	"ctr":7,
	"threads":[
		{
			"postCtr":0,
			"imageCtr":0,
			"id":3,
			"time":0,
			"image":{
				"fileType":0,
				"dims":[0,0,0,0],
				"size":0,
				"MD5":"",
				"SHA1":"foo",
				"name":""
			},
			"logCtr":33,
			"bumpTime":0,
			"replyTime":0,
			"board":"a"
		},
		{
			"postCtr":0,
			"imageCtr":0,
			"id":1,
			"time":0,
			"image":{
				"fileType":0,
				"dims":[0,0,0,0],
				"size":0,
				"MD5":"",
				"SHA1":"foo",
				"name":""
			},
			"logCtr":11,
			"bumpTime":0,
			"replyTime":0,
			"board":"a"
		}
	]
}`
	d.r.ServeHTTP(rec, req)
	assertBody(rec, removeIndentation(body), c)
	const etag = "W/7"
	assertEtag(rec, etag, c)

	// Etags match
	rec, req = newPair(c, "/json/a/")
	req.Header.Set("If-None-Match", etag)
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 304, c)
}

func (d *DB) TestAllBoardJSON(c *C) {
	setupPosts(c)

	const etag = "W/8"
	const body = `
{
	"ctr":8,
	"threads":[
		{
			"postCtr":0,
			"imageCtr":0,
			"id":4,
			"time":0,
			"image":{
				"fileType":0,
				"dims":[0,0,0,0],
				"size":0,
				"MD5":"",
				"SHA1":"foo",
				"name":""
			},
			"logCtr":44,
			"bumpTime":0,
			"replyTime":0,
			"board":"c"
		},
		{
			"postCtr":0,
			"imageCtr":0,
			"id":3,
			"time":0,
			"image":{
				"fileType":0,
				"dims":[0,0,0,0],
				"size":0,
				"MD5":"",
				"SHA1":"foo",
				"name":""
			},
			"logCtr":33,
			"bumpTime":0,
			"replyTime":0,
			"board":"a"
		},
		{
			"postCtr":0,
			"imageCtr":0,
			"id":1,
			"time":0,
			"image":{
				"fileType":0,
				"dims":[0,0,0,0],
				"size":0,
				"MD5":"",
				"SHA1":"foo",
				"name":""
			},
			"logCtr":11,
			"bumpTime":0,
			"replyTime":0,
			"board":"a"
		}
	]
}`
	rec, req := newPair(c, "/json/all/")
	d.r.ServeHTTP(rec, req)
	assertBody(rec, removeIndentation(body), c)
	assertEtag(rec, etag, c)

	// Etags match
	rec, req = newPair(c, "/json/all/")
	req.Header.Set("If-None-Match", etag)
	d.r.ServeHTTP(rec, req)
	allBoardJSON(rec, req)
	assertCode(rec, 304, c)
}

func (d *DB) TestThreadJSON(c *C) {
	setupPosts(c)

	// Invalid board
	rec, req := newPair(c, "/json/nope/1")
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 404, c)

	// Invalid post number
	rec, req = newPair(c, "/json/a/www")
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 400, c)

	// Non-existing thread
	rec, req = newPair(c, "/json/a/22")
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 404, c)

	// Valid thread request
	const body = `
{
	"postCtr":0,
	"imageCtr":0,
	"editing":false,
	"id":1,
	"time":0,
	"body":"",
	"image":{
		"fileType":0,
		"dims":[0,0,0,0],
		"size":0,
		"MD5":"",
		"SHA1":"foo",
		"name":""
	},
	"logCtr":11,
	"bumpTime":0,
	"replyTime":0,
	"board":"a",
	"posts":{
		"2":{
			"editing":false,
			"id":2,
			"time":0,
			"body":""
		}
	}
}`
	const etag = "W/11"
	rec, req = newPair(c, "/json/a/1")
	d.r.ServeHTTP(rec, req)
	assertBody(rec, removeIndentation(body), c)
	assertEtag(rec, etag, c)

	// Etags match
	rec, req = newPair(c, "/json/a/1")
	req.Header.Set("If-None-Match", etag)
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 304, c)
}

func (d *DB) TestServeBoardConfigsInvalidBoard(c *C) {
	(*config.Get()).Boards = []string{}
	rec, req := newPair(c, "/json/boardConfig/a")
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 404, c)
}

func (d *DB) TestServeBoardConfigs(c *C) {
	(*config.Get()).Boards = []string{"a"}
	conf := config.BoardConfigs{
		ID:       "a",
		CodeTags: true,
		Title:    "Animu",
		Notice:   "Notice",
		Banners:  []string{},
	}
	c.Assert(db.Write(r.Table("boards").Insert(conf)), IsNil)

	std, err := conf.MarshalPublicJSON()
	c.Assert(err, IsNil)

	rec, req := newPair(c, "/json/boardConfig/a")
	d.r.ServeHTTP(rec, req)
	assertBody(rec, string(std), c)
}

func (d *DB) TestServeAllBoardConfigs(c *C) {
	std := []byte("foo")
	config.AllBoardConfigs = std
	rec, req := newPair(c, "/json/boardConfig/all")
	d.r.ServeHTTP(rec, req)
	assertBody(rec, string(std), c)
}

func (d *DB) TestServeBoardList(c *C) {
	conf := []config.BoardConfigs{
		{
			ID:    "a",
			Title: "Animu",
		},
		{
			ID:    "g",
			Title: "Technology",
		},
	}
	c.Assert(db.Write(r.Table("boards").Insert(conf)), IsNil)

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

	rec, req := newPair(c, "/json/boardList")
	d.r.ServeHTTP(rec, req)
	assertBody(rec, std, c)
}

func (d *DB) TestServeBoardListNoBoards(c *C) {
	rec, req := newPair(c, "/json/boardList")
	d.r.ServeHTTP(rec, req)
	assertBody(rec, "[]", c)
}

func (d *DB) TestServeStaffPosition(c *C) {
	staff := map[string][]string{
		"owners": {"admin"},
	}
	boards := []config.BoardConfigs{
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
	}
	c.Assert(db.Write(r.Table("boards").Insert(boards)), IsNil)

	samples := [...]struct {
		position, user, res string
	}{
		{"owners", "admin", `["a","c"]`},
		{"mod", "admin", "[]"},
		{"owners", "bullshit", "[]"},
		{"bullocks", "bullshit", "[]"},
	}

	for _, s := range samples {
		path := fmt.Sprintf("/json/positions/%s/%s", s.position, s.user)
		rec, req := newPair(c, path)
		d.r.ServeHTTP(rec, req)
		assertCode(rec, 200, c)
		assertBody(rec, s.res, c)
	}
}

func (d *DB) TestServeBacklog(c *C) {
	log := [][]byte{
		[]byte("foo"),
		[]byte("bar"),
		[]byte("baz"),
	}
	thread := types.DatabaseThread{
		ID:  1,
		Log: log,
	}
	c.Assert(db.Write(r.Table("threads").Insert(thread)), IsNil)

	samples := [...]struct {
		url  string
		code int
		body string
	}{
		{"/1/0/1", 200, "foo"},
		{"/1/0/2", 200, "42foo\u0000bar"},
		{"/2/0/1", 200, ""},
		{"/a/0/1", 400, ""},
		{"/-1/0/1", 400, ""},
	}

	for _, s := range samples {
		rec, req := newPair(c, "/json/backlog"+s.url)
		d.r.ServeHTTP(rec, req)
		assertCode(rec, s.code, c)
		if s.code == 200 {
			assertBody(rec, s.body, c)
		}
	}
}

func (d *DB) TestSpoilerImageWrongPassword(c *C) {
	const password = "123"
	hash, err := auth.BcryptHash(password, 6)
	c.Assert(err, IsNil)

	thread := types.DatabaseThread{
		ID: 1,
		Posts: map[int64]types.DatabasePost{
			1: {
				Password: hash,
				Post: types.Post{
					ID: 1,
					Image: &types.Image{
						ImageCommon: types.ImageCommon{
							SHA1: "123",
						},
					},
				},
			},
			2: {
				Password: hash,
				Post: types.Post{
					ID: 1,
				},
			},
		},
	}
	c.Assert(db.Write(r.Table("threads").Insert(thread)), IsNil)

	samples := [...]struct {
		id        int64
		password  string
		code      int
		spoilered bool
	}{
		{2, password, 400, false}, // No image
		{1, "122", 403, false},    // Wrong password
		{1, password, 200, true},  // Success
		{1, password, 200, true},  // Already spoilered
	}

	for _, s := range samples {
		data := spoilerRequest{
			ID:       s.id,
			Password: s.password,
		}
		rec, req := newJSONPair(c, "/json/spoiler", data)
		d.r.ServeHTTP(rec, req)

		assertCode(rec, s.code, c)

		var spoilered bool
		msg := []byte("11" + strconv.Itoa(int(s.id)))
		q := r.And(
			db.FindParentThread(s.id).Field("log").Contains(msg),
			db.FindPost(s.id).Field("image").Field("spoiler"),
		)
		c.Assert(db.One(q, &spoilered), IsNil)
		c.Assert(spoilered, Equals, s.spoilered)
	}
}
