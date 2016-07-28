package websockets

import (
	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

func (*DB) TestNotAdmin(c *C) {
	cl := &Client{}
	cl.ID = "foo"
	for _, fn := range []handler{configServer} {
		c.Assert(fn(nil, cl), Equals, errAccessDenied)
	}
}

func (*DB) TestServerConfigRequest(c *C) {
	config.Set(config.Defaults)
	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	cl.UserID = "admin"

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
	cl.UserID = "admin"

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

func (*DB) TestInvalidBoardName(c *C) {
	for _, name := range [...]string{"abcd", "", ":^)"} {
		req := boardCreationRequest{
			Name:  name,
			Title: "foo",
		}
		assertLoggedInResponse(req, createBoard, "123", []byte("401"), c)
	}
}

func (*DB) TestBoardTitleTooLong(c *C) {
	title, err := auth.RandomID(101)
	c.Assert(err, IsNil)
	title = title[:101]
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
	assertLoggedInResponse(req, createBoard, "123", []byte("402"), c)
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
		Banners:   []string{},
		Staff: map[string][]string{
			"owners": []string{userID},
		},
	}
	c.Assert(board, DeepEquals, std)

	var boards []string
	c.Assert(db.All(db.GetMain("config").Field("boards"), &boards), IsNil)
	c.Assert(boards, DeepEquals, []string{"a"})
}

func (*DB) TestNotBoardOwner(c *C) {
	req := config.BoardConfigs{
		ID: "a",
	}
	cl := &Client{
		Ident: auth.Ident{
			UserID: "123",
		},
		sessionToken: "foo",
	}
	data := marshalJSON(req, c)
	c.Assert(configBoard(data, cl), Equals, errAccessDenied)
}

func (*DB) TestBoardConfiguration(c *C) {
	const (
		id    = "123"
		board = "a"
	)
	req := config.BoardConfigs{
		ID: board,
		PostParseConfigs: config.PostParseConfigs{
			ForcedAnon: true,
		},
		Eightball: []string{},
		Banners:   []string{},
		Staff:     map[string][]string{},
	}
	init := config.BoardConfigs{
		ID:        board,
		Eightball: []string{},
		Banners:   []string{},
		Staff: map[string][]string{
			"owners": []string{id},
		},
	}
	c.Assert(db.Write(r.Table("boards").Insert(init)), IsNil)

	assertLoggedInResponse(req, configBoard, id, []byte("41true"), c)

	var res config.BoardConfigs
	c.Assert(db.One(db.GetBoardConfig(board), &res), IsNil)
	c.Assert(res, DeepEquals, req)
}
