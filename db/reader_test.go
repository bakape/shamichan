package db

import (
	"reflect"
	"testing"

	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
)

func TestReader(t *testing.T) {
	assertTableClear(t, "posts", "threads", "boards", "main")

	assertInsert(t, "posts", []types.DatabasePost{
		{
			Post: types.Post{
				ID:    1,
				OP:    1,
				Board: "a",
			},
			Log: [][]byte{{1, 2, 3}},
		},
		{
			Post: types.Post{
				ID:    2,
				OP:    1,
				Board: "a",
				Body:  "foo",
			},
			Log: [][]byte{{3, 4, 5}},
		},
		{
			Post: types.Post{
				ID:    4,
				OP:    1,
				Board: "a",
			},
			Log: [][]byte{{1}},
		},
		{
			Post: types.Post{
				ID:    3,
				OP:    3,
				Board: "c",
			},
			Log: [][]byte{{1}, {2}},
		},
	})

	assertInsert(t, "threads", []types.DatabaseThread{
		{
			ID:      1,
			Board:   "a",
			PostCtr: 3,
		},
		{
			ID:      3,
			Board:   "c",
			PostCtr: 1,
		},
	})

	assertInsert(t, "main", []map[string]interface{}{
		{
			"id":      "info",
			"postCtr": 3,
		},
		{
			"id": "boardCtrs",
			"a":  2,
			"c":  1,
		},
	})

	t.Run("GetPost", testGetPost)
	t.Run("GetAllBoard", testGetAllBoard)
	t.Run("GetBoard", testGetBoard)
	t.Run("GetThread", testGetThread)
}

func testGetPost(t *testing.T) {
	t.Parallel()

	// Does not exist
	post, err := GetPost(99)
	if err != r.ErrEmptyResult {
		t.Errorf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(post, types.Post{}) {
		t.Errorf("post not empty: %#v", post)
	}

	// Valid read
	std := types.Post{
		ID:    2,
		OP:    1,
		Board: "a",
		Body:  "foo",
	}
	post, err = GetPost(2)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(post, std) {
		logUnexpected(t, std, post)
	}
}

func testGetAllBoard(t *testing.T) {
	t.Parallel()

	std := types.Board{
		Ctr: 3,
		Threads: []types.BoardThread{
			{
				ID:      1,
				PostCtr: 3,
				Board:   "a",
			},
			{
				ID:      3,
				PostCtr: 1,
				Board:   "c",
			},
		},
	}

	board, err := GetAllBoard()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(board, &std) {
		logUnexpected(t, &std, board)
	}
}

func testGetBoard(t *testing.T) {
	t.Parallel()

	cases := [...]struct {
		name, id string
		std      types.Board
	}{
		{
			name: "full",
			id:   "c",
			std: types.Board{
				Ctr: 1,
				Threads: []types.BoardThread{
					{
						ID:      3,
						PostCtr: 1,
						Board:   "c",
					},
				},
			},
		},
		{
			name: "empty",
			id:   "z",
			std: types.Board{
				Ctr:     0,
				Threads: nil,
			},
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			board, err := GetBoard(c.id)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(board, &c.std) {
				logUnexpected(t, &c.std, board)
			}
		})
	}
}

func testGetThread(t *testing.T) {
	t.Parallel()

	thread1 := types.Thread{
		PostCtr: 3,
		Post: types.Post{
			ID:     1,
			LogCtr: 1,
			Board:  "a",
		},
		Posts: []types.Post{
			{
				ID:     2,
				OP:     1,
				LogCtr: 1,
				Board:  "a",
				Body:   "foo",
			},
			{
				ID:     4,
				OP:     1,
				LogCtr: 1,
				Board:  "a",
			},
		},
	}
	sliced := thread1
	sliced.Posts = sliced.Posts[1:]

	cases := [...]struct {
		name  string
		id    int64
		lastN int
		std   *types.Thread
		err   error
	}{
		{
			name: "full",
			id:   1,
			std:  &thread1,
		},
		{
			name:  "last 1 reply",
			id:    1,
			lastN: 1,
			std:   &sliced,
		},
		{
			name: "no replies ;_;",
			id:   3,
			std: &types.Thread{
				PostCtr: 1,
				Post: types.Post{
					ID:     3,
					Board:  "c",
					LogCtr: 2,
				},
				Posts: []types.Post{},
			},
		},
		{
			name: "nonexistant thread",
			id:   99,
			err:  r.ErrEmptyResult,
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			thread, err := GetThread(c.id, c.lastN)
			if err != c.err {
				t.Fatalf("unexpected error: %#v", err)
			}
			if !reflect.DeepEqual(thread, c.std) {
				logUnexpected(t, c.std, thread)
			}
		})
	}
}

// func (*Tests) TestReaderGetThread(c *C) {
// 	// No replies ;_;
// 	std := &types.Thread{
// 		Board: "a",
// 		Post: types.Post{
// 			ID: 4,
// 		},
// 		Posts: map[int64]types.Post{},
// 	}
// 	thread, err := GetThread(4, 0)
// 	c.Assert(err, IsNil)
// 	c.Assert(thread, DeepEquals, std)

// 	// With replies
// 	std = &types.Thread{
// 		ImageCtr: 1,
// 		PostCtr:  2,
// 		LogCtr:   3,
// 		Board:    "a",
// 		Post: types.Post{
// 			ID: 1,
// 		},
// 		Posts: map[int64]types.Post{
// 			2: {
// 				ID: 2,
// 			},
// 			3: {
// 				ID: 3,
// 			},
// 		},
// 	}
// 	thread, err = GetThread(1, 0)
// 	c.Assert(err, IsNil)
// 	c.Assert(thread, DeepEquals, std)

// 	// Last 1 post
// 	delete(std.Posts, 2)
// 	thread, err = GetThread(1, 1)
// 	c.Assert(err, IsNil)
// 	c.Assert(thread, DeepEquals, std)
// }
