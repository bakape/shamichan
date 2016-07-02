package websockets

import (
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
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
