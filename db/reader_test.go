package db

import (
	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

var genericImage = types.Image{File: "foo"}

func (*DBSuite) TestNewReader(c *C) {
	ident := auth.Ident{}
	standard := &Reader{"a", ident}
	c.Assert(NewReader("a", ident), DeepEquals, standard)
}

func (*DBSuite) TestParsePost(c *C) {
	// Regular post
	r := NewReader("a", auth.Ident{})
	init := types.Post{
		Body:  "foo",
		Image: genericImage,
		IP:    "::1",
	}
	standard := types.Post{
		Body:  "foo",
		Image: genericImage,
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

func (*DBSuite) TestGetPost(c *C) {
	standard := types.Post{
		ID:    3,
		OP:    1,
		Board: "a",
	}
	samplePosts := []types.Post{
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
	}
	c.Assert(DB()(r.Table("posts").Insert(samplePosts)).Exec(), IsNil)
	sampleThreads := []types.Thread{
		{
			ID:      4,
			Deleted: true,
			Board:   "a",
		},
		{
			ID:    1,
			Board: "a",
		},
	}
	c.Assert(DB()(r.Table("threads").Insert(sampleThreads)).Exec(), IsNil)

	r := NewReader("a", auth.Ident{})
	empty := types.Post{}

	// Does not exist
	post, err := r.GetPost(7)
	c.Assert(err, IsNil)
	c.Assert(post, DeepEquals, empty)

	// Post deleted
	post, err = r.GetPost(2)
	c.Assert(err, IsNil)
	c.Assert(post, DeepEquals, empty)

	// Thread deleted
	post, err = r.GetPost(5)
	c.Assert(err, IsNil)
	c.Assert(post, DeepEquals, empty)

	// Board no longer accessable
	post, err = r.GetPost(8)
	c.Assert(err, IsNil)
	c.Assert(post, DeepEquals, empty)

	// Valid read
	post, err = r.GetPost(3)
	c.Assert(err, IsNil)
	c.Assert(post, DeepEquals, standard)
}

func (*DBSuite) TestGetJoinedThread(c *C) {
	// Only OP
	c.Assert(DB()(r.Table("threads").Insert(types.Thread{ID: 1})).Exec(), IsNil)
	samplePosts := types.Post{
		ID:    1,
		OP:    1,
		Image: genericImage,
	}
	c.Assert(DB()(r.Table("posts").Insert(samplePosts)).Exec(), IsNil)
	standard := joinedThread{
		Left: types.Thread{
			ID: 1,
		},
		Right: types.Post{
			ID:    1,
			Image: genericImage,
		},
	}

	thread, err := getJoinedThread(1)
	c.Assert(err, IsNil)
	c.Assert(thread, DeepEquals, standard)

	// 1 reply, no image
	sampleReply := types.Post{
		ID: 2,
		OP: 1,
	}
	c.Assert(DB()(r.Table("posts").Insert(sampleReply)).Exec(), IsNil)
	standard.Left.PostCtr++
	thread, err = getJoinedThread(1)
	c.Assert(err, IsNil)
	c.Assert(thread, DeepEquals, standard)

	// 2 replies, 1 image
	imagePost := types.Post{
		ID:    3,
		OP:    1,
		Image: genericImage,
	}
	c.Assert(DB()(r.Table("posts").Insert(imagePost)).Exec(), IsNil)
	standard.Left.PostCtr++
	standard.Left.ImageCtr++
	thread, err = getJoinedThread(1)
	c.Assert(err, IsNil)
	c.Assert(thread, DeepEquals, standard)
}

func (*DBSuite) TestParseThreads(c *C) {
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

func (*DBSuite) TestGetBoard(c *C) {
	setupPosts(c)
	standard := types.Board{
		Ctr: 7,
		Threads: []types.ThreadContainer{
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
		},
	}
	board, err := NewReader("a", auth.Ident{}).GetBoard()
	c.Assert(err, IsNil)
	c.Assert(board, DeepEquals, standard)
}

// Create a multipurpose set of threads and posts for tests
func setupPosts(c *C) {
	threads := []types.Thread{
		{ID: 1, Board: "a"},
		{ID: 3, Board: "a"},
		{ID: 4, Board: "c"},
	}
	c.Assert(DB()(r.Table("threads").Insert(threads)).Exec(), IsNil)

	posts := []types.Post{
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
	}
	c.Assert(DB()(r.Table("posts").Insert(posts)).Exec(), IsNil)

	main := []map[string]interface{}{
		{
			"id": "histCounts",
			"a":  7,
		},
		{
			"id":      "info",
			"postCtr": 8,
		},
	}
	c.Assert(DB()(r.Table("main").Insert(main)).Exec(), IsNil)
}

func (*DBSuite) TestGetAllBoard(c *C) {
	setupPosts(c)

	standard := types.Board{
		Ctr: 8,
		Threads: []types.ThreadContainer{
			{
				Thread: types.Thread{
					ID:      1,
					Board:   "a",
					PostCtr: 1,
				},
				Post: types.Post{
					ID:    1,
					Board: "a",
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
					ID:    4,
					Board: "c",
				},
				Post: types.Post{
					ID:    4,
					Board: "c",
					Image: genericImage,
				},
			},
		},
	}

	board, err := NewReader("a", auth.Ident{}).GetAllBoard()
	c.Assert(err, IsNil)
	c.Assert(board, DeepEquals, standard)
}

func (*DBSuite) TestReaderGetThread(c *C) {
	setupPosts(c)
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
	thread, err := rd.GetThread(3, 0)
	c.Assert(err, IsNil)
	c.Assert(thread, DeepEquals, standard)

	// With replies
	additional := types.Post{
		ID:    5,
		OP:    1,
		Board: "a",
		Image: genericImage,
	}
	c.Assert(DB()(r.Table("posts").Insert(additional)).Exec(), IsNil)
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
	thread, err = rd.GetThread(1, 0)
	c.Assert(err, IsNil)
	c.Assert(thread, DeepEquals, standard)

	// Last 1 post
	delete(standard.Posts, "2")
	thread, err = rd.GetThread(1, 1)
	c.Assert(err, IsNil)
	c.Assert(thread, DeepEquals, standard)
}
