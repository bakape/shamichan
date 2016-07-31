package websockets

import (
	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/imager"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

var (
	// JPEG sample image standard struct
	stdJPEG = types.ImageCommon{
		SHA1:     "012a2f912c9ee93ceb0ccb8684a29ec571990a94",
		FileType: 0,
		Dims:     [4]uint16{1, 1, 1, 1},
		MD5:      "60e41092581f7b329b057b8402caa8a7",
		Size:     300792,
	}
)

func (*DB) TestCreateThreadOnInvalidBoard(c *C) {
	req := types.ThreadCreationRequest{
		Board: "all",
	}
	c.Assert(insertThread(marshalJSON(req, c), new(Client)), Equals, errInvalidBoard)
}

func (*DB) TestCreateThreadOnReadOnlyBoard(c *C) {
	q := r.Table("boards").Insert(config.BoardConfigs{
		ID: "a",
		PostParseConfigs: config.PostParseConfigs{
			ReadOnly: true,
		},
	})
	c.Assert(db.Write(q), IsNil)

	req := types.ThreadCreationRequest{
		Board: "a",
	}
	err := insertThread(marshalJSON(req, c), new(Client))
	c.Assert(err, Equals, errReadOnly)
}

func (*DB) TestThreadCreation(c *C) {
	populateMainTable(c)
	writeGenericBoardConfig(c)
	c.Assert(db.Write(r.Table("images").Insert(stdJPEG)), IsNil)
	_, token, err := imager.NewImageToken(stdJPEG.SHA1)
	c.Assert(err, IsNil)

	sv := newWSServer(c)
	defer sv.Close()
	cl, wcl := sv.NewClient()
	cl.IP = "::1"

	std := types.DatabaseThread{
		ID:      6,
		Subject: "subject",
		Board:   "a",
		Posts: map[int64]types.DatabasePost{
			6: types.DatabasePost{
				IP: "::1",
				Post: types.Post{
					ID:   6,
					Body: "body",
					Name: "name",
					Image: &types.Image{
						Spoiler:     true,
						ImageCommon: stdJPEG,
						Name:        "foo",
					},
				},
			},
		},
		Log: [][]byte{},
	}

	req := types.ThreadCreationRequest{
		PostCredentials: types.PostCredentials{
			Name:     "name",
			Password: "123",
		},
		Subject:    "subject",
		Board:      "a",
		Body:       "body",
		ImageName:  "foo.jpeg",
		ImageToken: token,
		Spoiler:    true,
	}
	data := marshalJSON(req, c)
	c.Assert(insertThread(data, cl), IsNil)
	assertMessage(wcl, []byte("010"), c)

	var thread types.DatabaseThread
	c.Assert(db.One(r.Table("threads").Get(6), &thread), IsNil)

	// Pointers have to be dereferenced to be asserted
	c.Assert(*thread.Posts[6].Image, DeepEquals, *std.Posts[6].Image)

	// Normalize timestamps and pointer fields
	then := thread.BumpTime
	std.BumpTime = then
	std.ReplyTime = then
	post := std.Posts[6]
	post.Time = then
	post.Password = thread.Posts[6].Password
	post.Image = thread.Posts[6].Image
	std.Posts[6] = post

	c.Assert(thread, DeepEquals, std)
}

func populateMainTable(c *C) {
	mains := []map[string]interface{}{
		{
			"id":      "info",
			"postCtr": 5,
		},
		{
			"id": "boardCtrs",
		},
	}
	c.Assert(db.Write(r.Table("main").Insert(mains)), IsNil)
}

func writeGenericBoardConfig(c *C) {
	conf := config.BoardConfigs{
		ID: "a",
	}
	c.Assert(db.Write(r.Table("boards").Insert(conf)), IsNil)
}

func (*DB) TestTextOnlyThreadCreation(c *C) {
	populateMainTable(c)
	conf := config.BoardConfigs{
		ID: "a",
		PostParseConfigs: config.PostParseConfigs{
			TextOnly: true,
		},
	}
	c.Assert(db.Write(r.Table("boards").Insert(conf)), IsNil)

	sv := newWSServer(c)
	defer sv.Close()
	cl, _ := sv.NewClient()
	req := types.ThreadCreationRequest{
		PostCredentials: types.PostCredentials{
			Name:     "name",
			Password: "123",
		},
		Subject: "subject",
		Board:   "a",
		Body:    "body",
	}
	data := marshalJSON(req, c)
	c.Assert(insertThread(data, cl), IsNil)

	var post types.Post
	c.Assert(db.One(db.FindPost(6), &post), IsNil)
	c.Assert(post.Image, IsNil)
}

func (*DB) TestGetInvalidImage(c *C) {
	const (
		name  = "foo.jpeg"
		token = "dasdasd-ad--dsad-ads-d-ad-"
	)
	r128, err := auth.RandomID(128)
	c.Assert(err, IsNil)
	r128 = r128[:128]
	r201, err := auth.RandomID(201)
	c.Assert(err, IsNil)
	r201 = r201[:201]

	samples := [...]struct {
		token, name string
		err         error
	}{
		{"", name, errInvalidImageToken},
		{r128, name, errInvalidImageToken},
		{token, "", errNoImageName},
		{token, r201, errImageNameTooLong},
		{token, name, errInvalidImageToken}, // No token in the database
	}

	for _, s := range samples {
		_, err := getImage(s.token, s.name, false)
		c.Assert(err, Equals, s.err)
	}
}
