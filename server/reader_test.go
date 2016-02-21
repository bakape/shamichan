package server

import (
	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

var genericImage = types.Image{File: "foo"}

func (*DB) TestNewReader(c *C) {
	ident := auth.Ident{}
	standard := &Reader{"a", ident}
	c.Assert(NewReader("a", ident), DeepEquals, standard)
}

func (*DB) TestParsePost(c *C) {
	// Regular post
	r := NewReader("a", auth.Ident{})
	img := types.Image{File: "foo"}
	init := types.Post{
		Body:  "foo",
		Image: img,
		IP:    "::1",
	}
	standard := types.Post{
		Body:  "foo",
		Image: img,
	}
	p := init
	c.Assert(r.parsePost(p), DeepEquals, standard)

	// Image deleted
	p = init
	p.ImgDeleted = true
	c.Assert(r.parsePost(p), DeepEquals, types.Post{Body: "foo"})

	// Post deleted
	p = init
	p.Deleted = true
	c.Assert(r.parsePost(p), DeepEquals, types.Post{})
}

func (*DB) TestGetPost(c *C) {
	standard := types.Post{
		ID:    3,
		OP:    1,
		Board: "a",
	}
	db()(r.Table("posts").Insert([]types.Post{
		{
			ID:      2,
			OP:      1,
			Deleted: true,
			Board:   "a",
		},
		standard,
		{
			ID:    5,
			OP:    4,
			Board: "a",
		},
		{
			ID:    8,
			OP:    1,
			Board: "q",
		},
	})).Exec()
	db()(r.Table("threads").Insert([]types.Thread{
		{
			ID:      4,
			Deleted: true,
			Board:   "a",
		},
		{
			ID:    1,
			Board: "a",
		},
	})).Exec()
	r := NewReader("a", auth.Ident{})
	empty := types.Post{}
	c.Assert(r.GetPost(7), DeepEquals, empty)    // Does not exist
	c.Assert(r.GetPost(2), DeepEquals, empty)    // Post deleted
	c.Assert(r.GetPost(5), DeepEquals, empty)    // Thread deleted
	c.Assert(r.GetPost(8), DeepEquals, empty)    // Board no longer accessable
	c.Assert(r.GetPost(3), DeepEquals, standard) // Valid read
}

func (*DB) TestGetJoinedThread(c *C) {
	// Only OP
	db()(r.Table("threads").Insert(types.Thread{ID: 1})).Exec()
	db()(r.Table("posts").Insert(types.Post{
		ID:    1,
		OP:    1,
		Image: genericImage,
	})).Exec()
	standard := joinedThread{
		Left: types.Thread{
			ID: 1,
		},
		Right: types.Post{
			ID:    1,
			Image: genericImage,
		},
	}
	c.Assert(getJoinedThread(1), DeepEquals, standard)

	// 1 reply, no image
	db()(r.Table("posts").Insert(types.Post{
		ID: 2,
		OP: 1,
	})).Exec()

	standard.Left.PostCtr++
	c.Assert(getJoinedThread(1), DeepEquals, standard)

	// 2 replies, 1 image
	db()(r.Table("posts").Insert(types.Post{
		ID:    3,
		OP:    1,
		Image: genericImage,
	})).Exec()
	standard.Left.PostCtr++
	standard.Left.ImageCtr++
	c.Assert(getJoinedThread(1), DeepEquals, standard)
}

func (*DB) TestParseThreads(c *C) {
	threads := []joinedThread{
		{
			Left: types.Thread{
				ID:      2,
				Deleted: true,
			},
			Right: types.Post{
				ID: 2,
			},
		},
	}
	r := NewReader("a", auth.Ident{})

	// Zero length
	c.Assert(r.parseThreads(threads), DeepEquals, []types.ThreadContainer(nil))

	threads = append([]joinedThread{
		{
			Left: types.Thread{
				ID: 1,
			},
			Right: types.Post{
				ID: 1,
			},
		},
	}, threads...)
	standard := []types.ThreadContainer{
		{
			Thread: types.Thread{ID: 1},
			Post:   types.Post{ID: 1},
		},
	}
	c.Assert(r.parseThreads(threads), DeepEquals, standard)
}

func (*DB) TestGetBoard(c *C) {
	setupPosts()
	standard := types.Board{
		Ctr: 7,
		Threads: []types.ThreadContainer{
			{
				Thread: types.Thread{
					ID:    3,
					Board: "a",
				},
				Post: types.Post{
					ID:    3,
					Board: "a",
					Image: genericImage,
				},
			},
			{
				Thread: types.Thread{
					ID:      1,
					PostCtr: 1,
					Board:   "a",
				},
				Post: types.Post{
					ID:    1,
					Board: "a",
					Image: genericImage,
				},
			},
		},
	}
	c.Assert(NewReader("a", auth.Ident{}).GetBoard(), DeepEquals, standard)
}

// Create a multipurpose set of threads and posts for tests
func setupPosts() {
	db()(r.Table("threads").Insert([]types.Thread{
		{ID: 1, Board: "a"},
		{ID: 3, Board: "a"},
		{ID: 4, Board: "c"},
	})).Exec()
	db()(r.Table("posts").Insert([]types.Post{
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

	standard := types.Board{
		Ctr: 8,
		Threads: []types.ThreadContainer{
			{
				Thread: types.Thread{
					ID:    4,
					Board: "c",
				},
				Post: types.Post{
					ID:    4,
					Board: "c",
					Image: genericImage,
				},
			},
			{
				Thread: types.Thread{
					ID:    3,
					Board: "a",
				},
				Post: types.Post{
					ID:    3,
					Board: "a",
					Image: genericImage,
				},
			},
			{
				Thread: types.Thread{
					ID:      1,
					PostCtr: 1,
					Board:   "a",
				},
				Post: types.Post{
					ID:    1,
					Board: "a",
					Image: genericImage,
				},
			},
		},
	}
	c.Assert(NewReader("a", auth.Ident{}).GetAllBoard(), DeepEquals, standard)
}

func (*DB) TestReaderGetThread(c *C) {
	setupPosts()
	rd := NewReader("a", auth.Ident{})

	// No replies ;_;
	standard := types.ThreadContainer{
		Thread: types.Thread{
			ID:    3,
			Board: "a",
		},
		Post: types.Post{
			ID:    3,
			Board: "a",
			Image: genericImage,
		},
	}
	c.Assert(rd.GetThread(3, 0), DeepEquals, standard)

	// With replies
	additional := types.Post{
		ID:    5,
		OP:    1,
		Board: "a",
		Image: genericImage,
	}
	db()(r.Table("posts").Insert(additional)).Exec()
	standard = types.ThreadContainer{
		Thread: types.Thread{
			ID:       1,
			Board:    "a",
			PostCtr:  2,
			ImageCtr: 1,
		},
		Post: types.Post{
			ID:    1,
			Board: "a",
			Image: genericImage,
		},
		Posts: map[string]types.Post{
			"2": {
				ID:    2,
				OP:    1,
				Board: "a",
			},
			"5": additional,
		},
	}
	c.Assert(rd.GetThread(1, 0), DeepEquals, standard)

	// Last 1 post
	delete(standard.Posts, "2")
	c.Assert(rd.GetThread(1, 1), DeepEquals, standard)
}
