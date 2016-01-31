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
	// Only OP
	db()(r.Table("threads").Insert(Thread{ID: 1})).Exec()
	db()(r.Table("posts").Insert(Post{
		ID:    1,
		OP:    1,
		Image: Image{Src: "Foo"},
	})).Exec()
	standard := joinedThread{
		Left: Thread{
			ID: 1,
		},
		Right: Post{
			ID:    1,
			Image: Image{Src: "Foo"},
		},
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

func (*DB) TestParseThreads(c *C) {
	threads := []joinedThread{
		{
			Left: Thread{
				ID: 1,
			},
			Right: Post{
				ID: 1,
			},
		},
		{
			Left: Thread{
				ID:      2,
				Deleted: true,
			},
			Right: Post{
				ID: 2,
			},
		},
	}
	setupBoardAccess()

	// Can't see deleted threads
	standard := []ThreadContainer{
		{
			Thread: Thread{ID: 1},
			Post:   Post{ID: 1},
		},
	}
	r := NewReader("a", Ident{})
	c.Assert(r.parseThreads(threads), DeepEquals, standard)

	// Can see deleted threads
	r = NewReader("a", Ident{Auth: "admin"})
	standard = append(standard, ThreadContainer{
		Thread: Thread{
			Deleted: true,
			ID:      2,
		},
		Post: Post{
			ID: 2,
		},
	})
	c.Assert(r.parseThreads(threads), DeepEquals, standard)
}

var genericImage = Image{Src: "foo"}

func (*DB) TestGetBoard(c *C) {
	setupPosts()
	setupBoardAccess()
	standard := Board{
		Ctr: 7,
		Threads: []ThreadContainer{
			{
				Thread: Thread{
					ID:    3,
					Board: "a",
				},
				Post: Post{
					ID:    3,
					Board: "a",
					Image: genericImage,
				},
			},
			{
				Thread: Thread{
					ID:      1,
					PostCtr: 1,
					Board:   "a",
				},
				Post: Post{
					ID:    1,
					Board: "a",
					Image: genericImage,
				},
			},
		},
	}
	c.Assert(NewReader("a", Ident{}).GetBoard(), DeepEquals, standard)
}

// Create a multipurpose set of threads and posts for tests
func setupPosts() {
	db()(r.Table("threads").Insert([]Thread{
		{ID: 1, Board: "a"},
		{ID: 3, Board: "a"},
		{ID: 4, Board: "c"},
		{ID: 5, Board: "staff"},
	})).Exec()
	db()(r.Table("posts").Insert([]Post{
		{
			ID:    1,
			OP:    1,
			Board: "a",
			Image: genericImage,
		},
		{
			ID:    2,
			OP:    1,
			Board: "a",
		},
		{
			ID:    3,
			OP:    3,
			Board: "a",
			Image: genericImage,
		},
		{
			ID:    4,
			OP:    4,
			Board: "c",
			Image: genericImage,
		},
		{
			ID:    5,
			OP:    5,
			Board: "staff",
			Image: genericImage,
		},
	})).Exec()
	db()(r.Table("main").Insert(map[string]interface{}{
		"id": "histCounts",
		"a":  7,
	})).Exec()
	db()(r.Table("main").Insert(map[string]interface{}{
		"id":      "info",
		"postCtr": 8,
	})).Exec()
}

func (*DB) TestGetAllBoard(c *C) {
	setupPosts()
	setupBoardAccess()

	// Can't access staff board
	standard := Board{
		Ctr: 8,
		Threads: []ThreadContainer{
			{
				Thread: Thread{
					ID:    4,
					Board: "c",
				},
				Post: Post{
					ID:    4,
					Board: "c",
					Image: genericImage,
				},
			},
			{
				Thread: Thread{
					ID:    3,
					Board: "a",
				},
				Post: Post{
					ID:    3,
					Board: "a",
					Image: genericImage,
				},
			},
			{
				Thread: Thread{
					ID:      1,
					PostCtr: 1,
					Board:   "a",
				},
				Post: Post{
					ID:    1,
					Board: "a",
					Image: genericImage,
				},
			},
		},
	}
	c.Assert(NewReader("a", Ident{}).GetAllBoard(), DeepEquals, standard)

	// Can access staff board
	standard.Threads = append(
		standard.Threads[:1],
		append(
			[]ThreadContainer{
				{
					Thread: Thread{
						ID:    5,
						Board: "staff",
					},
					Post: Post{
						ID:    5,
						Board: "staff",
						Image: genericImage,
					},
				},
			},
			standard.Threads[1:]...,
		)...,
	)
	c.Assert(
		NewReader("a", Ident{Auth: "admin"}).GetAllBoard(),
		DeepEquals,
		standard,
	)
}
