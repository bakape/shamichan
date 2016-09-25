package db

import (
	"time"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

func (*Tests) TestSessionCleanup(c *C) {
	expired := time.Now().Add(-time.Hour)
	samples := []auth.User{
		{
			ID: "1",
			Sessions: []auth.Session{
				{
					Token:   "foo",
					Expires: expired,
				},
				{
					Token:   "bar",
					Expires: time.Now().Add(time.Hour),
				},
			},
		},
		{
			ID: "2",
			Sessions: []auth.Session{
				{
					Token:   "baz",
					Expires: expired,
				},
			},
		},
	}
	c.Assert(Write(r.Table("accounts").Insert(samples)), IsNil)

	c.Assert(expireUserSessions(), IsNil)

	var res1 []auth.Session
	c.Assert(All(GetAccount("1").Field("sessions"), &res1), IsNil)
	c.Assert(len(res1), Equals, 1)
	c.Assert(res1[0].Token, Equals, "bar")

	var res2 []auth.Session
	c.Assert(All(GetAccount("2").Field("sessions"), &res1), IsNil)
	c.Assert(res2, DeepEquals, []auth.Session(nil))
}

func (*Tests) TestOpenPostClosing(c *C) {
	thread := types.DatabaseThread{
		ID: 1,
		Posts: map[int64]types.DatabasePost{
			1: {
				Post: types.Post{
					ID:      1,
					Editing: true,
					Time:    time.Now().Add(-time.Minute * 31).Unix(),
				},
			},
			2: {
				Post: types.Post{
					ID:      2,
					Editing: true,
					Time:    time.Now().Unix(),
				},
			},
			3: {
				Post: types.Post{
					ID:      3,
					Editing: true,
					Time:    time.Now().Add(-time.Minute * 31).Unix(),
				},
			},
		},
		Log: [][]byte{[]byte{1, 22, 3}},
	}
	c.Assert(Write(r.Table("threads").Insert(thread)), IsNil)

	c.Assert(closeDanglingPosts(), IsNil)

	var log [][]byte
	c.Assert(All(r.Table("threads").Get(1).Field("log"), &log), IsNil)
	c.Assert(log, DeepEquals, append(thread.Log, []byte("061"), []byte("063")))

	samples := [...]struct {
		id      int64
		editing bool
	}{
		{1, false},
		{2, true},
		{3, false},
	}
	for _, s := range samples {
		var res bool
		c.Assert(One(FindPost(s.id).Field("editing"), &res), IsNil)
		c.Assert(res, DeepEquals, s.editing)
	}
}

func (*Tests) TestTokenExpiry(c *C) {
	const SHA1 = "123"
	img := types.ProtoImage{
		ImageCommon: types.ImageCommon{
			SHA1:     "123",
			FileType: types.JPEG,
		},
		Posts: 7,
	}
	c.Assert(Write(r.Table("images").Insert(img)), IsNil)

	expired := time.Now().Add(-time.Minute)
	tokens := [...]allocationToken{
		{
			SHA1:    SHA1,
			Expires: expired,
		},
		{
			SHA1:    SHA1,
			Expires: expired,
		},
		{
			SHA1:    SHA1,
			Expires: time.Now().Add(time.Minute),
		},
	}
	c.Assert(Write(r.Table("imageTokens").Insert(tokens)), IsNil)

	c.Assert(expireImageTokens(), IsNil)
	var posts int
	c.Assert(One(GetImage(SHA1).Field("posts"), &posts), IsNil)
	c.Assert(posts, Equals, 5)
}

func (*Tests) TestTokenExpiryNoTokens(c *C) {
	c.Assert(expireImageTokens(), IsNil)
}

func (*Tests) TestDeleteThreadWithoutImages(c *C) {
	thread := types.DatabaseThread{
		ID: 1,
		Posts: map[int64]types.DatabasePost{
			1: {
				Post: types.Post{
					ID: 1,
				},
			},
			2: {
				Post: types.Post{
					ID: 2,
				},
			},
		},
	}
	c.Assert(Write(r.Table("threads").Insert(thread)), IsNil)

	c.Assert(DeleteThread(1), IsNil)
	assertThreadDeleted(1, c)
}

func assertThreadDeleted(id int64, c *C) {
	var noThread bool
	q := r.Table("threads").Get(id).Eq(nil)
	c.Assert(One(q, &noThread), IsNil)
	c.Assert(noThread, Equals, true)
}

func (*Tests) TestDeleteNonExistantThread(c *C) {
	c.Assert(DeleteThread(1), IsNil)
}

func (*Tests) TestDeleteThread(c *C) {
	images := [...]types.ProtoImage{
		{
			ImageCommon: types.ImageCommon{
				SHA1: "2",
			},
			Posts: 2,
		},
		{
			ImageCommon: types.ImageCommon{
				SHA1: "3",
			},
			Posts: 3,
		},
	}
	c.Assert(Write(r.Table("images").Insert(images)), IsNil)

	thread := types.DatabaseThread{
		ID: 1,
		Posts: map[int64]types.DatabasePost{
			1: {
				Post: types.Post{
					ID: 1,
					Image: &types.Image{
						ImageCommon: types.ImageCommon{
							SHA1: "2",
						},
					},
				},
			},
			2: {
				Post: types.Post{
					ID: 2,
					Image: &types.Image{
						ImageCommon: types.ImageCommon{
							SHA1: "3",
						},
					},
				},
			},
		},
	}
	c.Assert(Write(r.Table("threads").Insert(thread)), IsNil)

	c.Assert(DeleteThread(1), IsNil)

	assertThreadDeleted(1, c)
	assertImageRefCount("2", 1, c)
	assertImageRefCount("3", 2, c)
}
