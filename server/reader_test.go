package server

import (
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

const (
	testSalt          = "r088PUX0qpUjhUyZby6e4pQcDh3zzUQUpeLOy7Hb"
	localhostMnemonic = "tyalitara"
)

func (*DB) TestNewReader(c *C) {
	setupBoardAccess()
	ident := Ident{}
	const board = "a"
	standard := &Reader{board, ident, false, false}
	r := NewReader(board, ident)
	c.Assert(r, DeepEquals, standard)
	ident.Auth = "admin"
	standard = &Reader{board, ident, true, true}
	r = NewReader(board, ident)
	c.Assert(r, DeepEquals, standard)
}

func (*DB) TestParsePost(c *C) {
	// Regular post
	setupBoardAccess()
	r := NewReader("a", Ident{})
	img := Image{Src: "foo"}
	mod := ModerationList{"a", "b"}
	init := Post{
		Body:  "foo",
		Image: img,
		Mod:   mod,
		IP:    "::1",
	}
	standard := Post{
		Body:  "foo",
		Image: img,
	}
	p := init
	c.Assert(r.parsePost(p), DeepEquals, standard)

	// Image deleted
	p = init
	p.ImgDeleted = true
	c.Assert(r.parsePost(p), DeepEquals, Post{Body: "foo"})

	// Post deleted
	p = init
	p.Deleted = true
	c.Assert(r.parsePost(p), DeepEquals, Post{})

	// Can see moderation
	r = NewReader("a", Ident{Auth: "janitor"})
	init.ImgDeleted = true
	init.Deleted = true
	p = init
	standard = Post{
		Body:       "foo",
		Image:      img,
		ImgDeleted: true,
		Deleted:    true,
		Mod:        mod,
	}
	c.Assert(r.parsePost(p), DeepEquals, standard)

	// Can see mnemonics
	r = NewReader("a", Ident{Auth: "admin"})
	standard.Mnemonic = localhostMnemonic
	p = init
	c.Assert(r.parsePost(p), DeepEquals, standard)
}

func (*DB) TestGetPost(c *C) {
	p := Post{
		ID:      2,
		OP:      1,
		Deleted: true,
		IP:      "::1",
	}
	db()(r.Table("posts").Insert(p)).Exec()
	setupBoardAccess()
	r := NewReader("a", Ident{})
	empty := Post{}

	// Does not exist
	c.Assert(r.GetPost(3), DeepEquals, empty)

	// Can not access
	c.Assert(r.GetPost(2), DeepEquals, empty)

	// Valid read
	standard := Post{
		ID:       2,
		OP:       1,
		Deleted:  true,
		Mnemonic: localhostMnemonic,
	}
	res := NewReader("a", Ident{Auth: "admin"}).GetPost(2)
	c.Assert(res, DeepEquals, standard)
}

func (*DB) TestGetJoinedThread(c *C) {
	op := Post{
		ID: 1,
		OP: 1,
		Image: Image{
			Src: "Foo",
		},
	}

	// Only OP
	db()(r.Table("threads").Insert(Thread{ID: 1})).Exec()
	db()(r.Table("posts").Insert(op)).Exec()
	standard := joinedThread{
		Left: Thread{
			ID:       1,
			PostCtr:  0,
			ImageCtr: 0,
		},
		Right: op,
	}
	c.Assert(getJoinedThread(1), DeepEquals, standard)

	// 1 reply, no image
	db()(r.Table("posts").Insert(Post{
		ID: 2,
		OP: 1,
	})).Exec()


	standard.Left.PostCtr++ 
	c.Assert(getJoinedThread(1), DeepEquals, standard)

	// 2 replies, 1 image
	db()(r.Table("posts").Insert(Post{
		ID:    3,
		OP:    1,
		Image: Image{Src: "foo"},
	})).Exec()
	standard.Left.PostCtr++
	standard.Left.ImageCtr++
	c.Assert(getJoinedThread(1), DeepEquals, standard)
}
