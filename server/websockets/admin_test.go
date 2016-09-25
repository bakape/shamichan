package websockets

import (
	"time"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

func (*DB) TestNotAdmin(c *C) {
	cl := &Client{}
	cl.UserID = "foo"
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

	c.Assert(configServer([]byte{}, cl), IsNil)
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

	var board config.DatabaseBoardConfigs
	c.Assert(db.One(db.GetBoardConfig(id), &board), IsNil)
	std := config.DatabaseBoardConfigs{
		BoardConfigs: config.BoardConfigs{
			ID:        id,
			Spoiler:   "default.jpg",
			Title:     title,
			Eightball: config.EightballDefaults,
			Banners:   []string{},
			Staff: map[string][]string{
				"owners": []string{userID},
			},
		},
	}
	c.Assert(board.Created.Before(time.Now()), Equals, true)
	c.Assert(board.BoardConfigs, DeepEquals, std.BoardConfigs)

	var boards []string
	c.Assert(db.All(db.GetMain("config").Field("boards"), &boards), IsNil)
	c.Assert(boards, DeepEquals, []string{"a"})
}
