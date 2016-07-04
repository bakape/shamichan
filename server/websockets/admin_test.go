package websockets

import (
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/util"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

func (*DB) TestNotAdmin(c *C) {
	cl := &Client{
		userID: "foo",
	}
	for _, fn := range []handler{configServer} {
		c.Assert(fn(nil, cl), Equals, errAccessDenied)
	}
}

func (*DB) TestServerConfigRequest(c *C) {
	config.Set(config.Defaults)
	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	cl.userID = "admin"

	c.Assert(configServer([]byte("null"), cl), IsNil)
	msg, err := encodeMessage(messageConfigServer, config.Get())
	c.Assert(err, IsNil)
	assertMessage(wcl, msg, c)
}

func (*DB) TestServerConfigSetting(c *C) {
	init := db.ConfigDocument{
		Document: db.Document{
			ID: "config",
		},
		Configs: config.Defaults,
	}
	c.Assert(db.Write(r.Table("main").Insert(init)), IsNil)

	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	cl.userID = "admin"

	req := config.Defaults
	req.Boards = []string{"fa"}
	req.DefaultCSS = "ashita"
	c.Assert(configServer(marshalJSON(req, c), cl), IsNil)
	assertMessage(wcl, []byte("39true"), c)

	var conf config.Configs
	c.Assert(db.One(db.GetMain("config"), &conf), IsNil)
	std := config.Defaults
	std.DefaultCSS = "ashita"
	c.Assert(conf, DeepEquals, std)
}

func (*DB) TestBpardNameTooLong(c *C) {
	req := boardCreationRequest{
		Name:  "abcd",
		Title: "foo",
	}
	assertLoggedInResponse(req, createBoard, "123", []byte("402"), c)
}

func (*DB) TestBoardTitleTooLong(c *C) {
	title, err := util.RandomID(101)
	c.Assert(err, IsNil)
	req := boardCreationRequest{
		Name:  "a",
		Title: title,
	}
	assertLoggedInResponse(req, createBoard, "123", []byte("403"), c)
}

func (*DB) TestBoardNameTaken(c *C) {
	q := r.Table("boards").Insert(db.Document{ID: "a"})
	c.Assert(db.Write(q), IsNil)
	req := boardCreationRequest{
		Name:  "a",
		Title: "/a/ - Animu & Mango",
	}
	assertLoggedInResponse(req, createBoard, "123", []byte("401"), c)
}

func (*DB) TestBoardCreation(c *C) {
	const (
		id     = "a"
		userID = "123"
		title  = "/a/ - Animu & Mango"
	)

	conf := db.ConfigDocument{
		Document: db.Document{ID: "config"},
		Configs:  config.Defaults,
	}
	c.Assert(db.Write(r.Table("main").Insert(conf)), IsNil)

	req := boardCreationRequest{
		Name:  id,
		Title: title,
	}
	assertLoggedInResponse(req, createBoard, userID, []byte("400"), c)

	var board config.BoardConfigs
	c.Assert(db.One(db.GetBoardConfig(id), &board), IsNil)
	std := config.BoardConfigs{
		ID:        id,
		Spoiler:   "default.jpg",
		Title:     title,
		Eightball: config.EightballDefaults,
		Staff: map[string][]string{
			"owners": []string{userID},
		},
	}
	c.Assert(board, DeepEquals, std)

	var boards []string
	c.Assert(db.All(db.GetMain("config").Field("boards"), &boards), IsNil)
	c.Assert(boards, DeepEquals, []string{"a"})
}
