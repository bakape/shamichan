package db

import (
	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

var (
	genericImage = types.Image{File: "foo"}

	sampleThreads = []types.DatabaseThread{
		{
			ID:       1,
			Board:    "a",
			ImageCtr: 1,
			PostCtr:  2,
			Posts: map[string]types.Post{
				"1": {
					ID:    1,
					Image: genericImage,
				},
				"2": {
					ID:    2,
					Image: genericImage,
				},
				"3": {
					ID: 3,
				},
			},
			Log: [][]byte{
				{1, 3, 4},
				{1, 3, 2},
				{1},
			},
		},
		{
			ID:    4,
			Board: "a",
			Posts: map[string]types.Post{
				"4": {
					ID:    4,
					Image: genericImage,
				},
			},
		},
		{
			ID:    5,
			Board: "c",
			Posts: map[string]types.Post{
				"5": {
					ID:    5,
					Image: genericImage,
				},
			},
		},
	}

	boardStandard = types.Board{
		Ctr: 7,
		Threads: []types.Thread{
			{
				Board: "a",
				Post: types.Post{
					ID:    4,
					Image: genericImage,
				},
				Posts: nil,
			},
			{
				ImageCtr: 1,
				PostCtr:  2,
				LogCtr:   3,
				Board:    "a",
				Post: types.Post{
					ID:    1,
					Image: genericImage,
				},
				Posts: nil,
			},
		},
	}
)

func (*DBSuite) TestNewReader(c *C) {
	ident := auth.Ident{}
	standard := &Reader{ident}
	c.Assert(NewReader(ident), DeepEquals, standard)
}

func (*DBSuite) TestParsePost(c *C) {
	// Regular post
	r := NewReader(auth.Ident{})
	standard := types.Post{
		Body:  "foo",
		Image: genericImage,
	}
	p := standard
	c.Assert(r.parsePost(p), DeepEquals, standard)

	// Image deleted
	p = standard
	p.ImgDeleted = true
	c.Assert(r.parsePost(p), DeepEquals, types.Post{Body: "foo"})

	// Post deleted
	p = standard
	p.Deleted = true
	c.Assert(r.parsePost(p), DeepEquals, types.Post{})
}

func (*DBSuite) TestGetPost(c *C) {
	conf := config.ServerConfigs{}
	conf.Boards.Enabled = []string{"a"}
	config.Set(conf)
	std := types.Post{ID: 2}
	threads := []types.DatabaseThread{
		{
			ID:    1,
			Board: "a",
			Posts: map[string]types.Post{
				"2": std,
				"3": {
					ID:      3,
					Deleted: true,
				},
			},
		},
		{
			ID:    4,
			Board: "q",
			Posts: map[string]types.Post{
				"5": {
					ID: 5,
				},
			},
		},
	}
	c.Assert(DB(r.Table("threads").Insert(threads)).Exec(), IsNil)

	rd := NewReader(auth.Ident{})

	empties := [...]struct {
		id, op int64
		board  string
	}{
		{2, 1, "c"},  // Wrong board
		{2, 76, "a"}, // Thread does not exist
		{8, 1, "a"},  // Post does not exist
		{3, 1, "a"},  // Post deleted
		{4, 5, "q"},  // Board no longer accessable
	}

	for _, args := range empties {
		post, err := rd.GetPost(args.id, args.op, args.board)
		c.Assert(err, IsNil)
		c.Assert(post, DeepEquals, types.Post{})
	}

	// Valid read
	post, err := rd.GetPost(2, 1, "a")
	c.Assert(err, IsNil)
	c.Assert(post, DeepEquals, std)
}

func (*DBSuite) TestParseThreads(c *C) {
	threads := []types.Thread{
		{
			Post: types.Post{
				ID:      2,
				Deleted: true,
			},
			Posts: map[string]types.Post{
				"2": {
					ID: 2,
				},
			},
		},
	}
	r := NewReader(auth.Ident{})

	// Zero length
	c.Assert(r.parseThreads(threads), DeepEquals, []types.Thread{})

	std := types.Thread{
		Post: types.Post{
			ID: 1,
		},
		Posts: map[string]types.Post{
			"1": {
				ID: 1,
			},
		},
	}
	threads = append(threads, std)
	c.Assert(r.parseThreads(threads), DeepEquals, []types.Thread{std})
}

func (*DBSuite) TestGetBoard(c *C) {
	setEnabledBoards("a")
	c.Assert(DB(r.Table("threads").Insert(sampleThreads)).Exec(), IsNil)

	boardCounters := map[string]interface{}{
		"id": "histCounts",
		"a":  7,
	}
	c.Assert(DB(r.Table("main").Insert(boardCounters)).Exec(), IsNil)

	board, err := NewReader(auth.Ident{}).GetBoard("a")
	c.Assert(err, IsNil)
	c.Assert(board, DeepEquals, &boardStandard)
}

func setEnabledBoards(boards ...string) {
	conf := config.ServerConfigs{}
	conf.Boards.Enabled = boards
	config.Set(conf)
}

func (*DBSuite) TestGetAllBoard(c *C) {
	setEnabledBoards("a")
	c.Assert(DB(r.Table("threads").Insert(sampleThreads)).Exec(), IsNil)
	info := infoDocument{
		Document: Document{"info"},
		PostCtr:  7,
	}
	c.Assert(DB(r.Table("main").Insert(info)).Exec(), IsNil)

	board, err := NewReader(auth.Ident{}).GetAllBoard()
	c.Assert(err, IsNil)
	c.Assert(board, DeepEquals, &boardStandard)
}

func (*DBSuite) TestReaderGetThread(c *C) {
	conf := config.ServerConfigs{}
	conf.Boards.Enabled = []string{"a"}
	config.Set(conf)
	c.Assert(DB(r.Table("threads").Insert(sampleThreads)).Exec(), IsNil)
	rd := NewReader(auth.Ident{})

	// No replies ;_;
	std := &types.Thread{
		Board: "a",
		Post: types.Post{
			ID:    4,
			Image: genericImage,
		},
		Posts: map[string]types.Post{},
	}
	thread, err := rd.GetThread(4, 0)
	c.Assert(err, IsNil)
	c.Assert(thread, DeepEquals, std)

	// With replies
	std = &types.Thread{
		Board:    "a",
		ImageCtr: 1,
		PostCtr:  2,
		LogCtr:   3,
		Post: types.Post{
			ID:    1,
			Image: genericImage,
		},
		Posts: map[string]types.Post{
			"2": {
				ID:    2,
				Image: genericImage,
			},
			"3": {
				ID: 3,
			},
		},
	}
	thread, err = rd.GetThread(1, 0)
	c.Assert(err, IsNil)
	c.Assert(thread, DeepEquals, std)

	// Last 1 post
	delete(std.Posts, "2")
	thread, err = rd.GetThread(1, 1)
	c.Assert(err, IsNil)
	c.Assert(thread, DeepEquals, std)
}
