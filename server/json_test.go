package server

import (
	"net/http"
	"strings"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/types"
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
	for _, table := range db.AllTables {
		c.Assert(db.Exec(r.Table(table).Delete()), IsNil)
	}
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
			Posts: map[int64]types.Post{
				1: {
					ID:    1,
					Image: genericImage,
				},
				2: {
					ID: 2,
				},
			},
		},
		{
			ID:    3,
			Board: "a",
			Log:   dummyLog(33),
			Posts: map[int64]types.Post{
				3: {
					ID:    3,
					Image: genericImage,
				},
			},
		},
		{
			ID:    4,
			Board: "c",
			Log:   dummyLog(44),
			Posts: map[int64]types.Post{
				4: {
					ID:    4,
					Image: genericImage,
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
	// No lastN query string
	req := newRequest(c, "/a/1")
	c.Assert(detectLastN(req), Equals, 0)

	// ?lastN value within bounds
	req = newRequest(c, "/a/1?lastN=100")
	c.Assert(detectLastN(req), Equals, 100)

	// ?lastN value beyond max
	req = newRequest(c, "/a/1?lastN=1000")
	c.Assert(detectLastN(req), Equals, 0)
}

func (d *DB) TestServePost(c *C) {
	setupPosts(c)

	// Invalid post number
	rec, req := newPair(c, "/json/post/www")
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 404, c)

	// Non-existing post or otherwise invalid post
	rec, req = newPair(c, "/json/post/66")
	d.r.ServeHTTP(rec, req)
	assertCode(rec, 404, c)

	// Existing post
	const (
		etag = "998db21ac97653d1"
		body = `{"editing":false,"id":2,"time":0,"body":"","op":1,"board":"a"}`
	)
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
			"editing":false,
			"id":3,
			"time":0,
			"body":"",
			"image":{
				"fileType":0,
				"dims":[0,0,0,0],
				"size":0,
				"MD5":"",
				"SHA1":"foo",
				"imgnm":""
			},
			"logCtr":33,
			"bumpTime":0,
			"replyTime":0,
			"board":"a"
		},
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
				"imgnm":""
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
			"editing":false,
			"id":4,
			"time":0,
			"body":"",
			"image":{
				"fileType":0,
				"dims":[0,0,0,0],
				"size":0,
				"MD5":"",
				"SHA1":"foo",
				"imgnm":""
			},
			"logCtr":44,
			"bumpTime":0,
			"replyTime":0,
			"board":"c"
		},
		{
			"postCtr":0,
			"imageCtr":0,
			"editing":false,
			"id":3,
			"time":0,
			"body":"",
			"image":{
				"fileType":0,
				"dims":[0,0,0,0],
				"size":0,
				"MD5":"",
				"SHA1":"foo",
				"imgnm":""
			},
			"logCtr":33,
			"bumpTime":0,
			"replyTime":0,
			"board":"a"
		},
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
				"imgnm":""
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
	assertCode(rec, 404, c)

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
		"imgnm":""
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
