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
			StandalonePost: types.StandalonePost{
				Post: types.Post{
					ID: 1,
				},
				OP:    1,
				Board: "a",
			},
			LastUpdated: 1,
			Log:         [][]byte{{1, 2, 3}},
		},
		{
			StandalonePost: types.StandalonePost{

				Post: types.Post{
					ID:   2,
					Body: "foo",
				},
				OP:    1,
				Board: "a",
			},
			LastUpdated: 2,
			Log:         [][]byte{{3, 4, 5}},
		},
		{
			StandalonePost: types.StandalonePost{
				Post: types.Post{
					ID: 4,
				},
				OP:    1,
				Board: "a",
			},
			LastUpdated: 3,
			Log:         [][]byte{{1}},
		},
		{
			StandalonePost: types.StandalonePost{
				Post: types.Post{
					ID: 3,
				},
				OP:    3,
				Board: "c",
			},
			LastUpdated: 4,
			Log:         [][]byte{{1}, {2}},
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
	if !reflect.DeepEqual(post, types.StandalonePost{}) {
		t.Errorf("post not empty: %#v", post)
	}

	// Valid read
	std := types.StandalonePost{
		Post: types.Post{
			ID:   2,
			Body: "foo",
		},
		OP:    1,
		Board: "a",
	}
	post, err = GetPost(2)
	if err != nil {
		t.Fatal(err)
	}
	assertDeepEquals(t, post, std)
}

func testGetAllBoard(t *testing.T) {
	t.Parallel()

	std := types.Board{
		Ctr: 3,
		Threads: []types.BoardThread{
			{
				ID:          1,
				PostCtr:     3,
				Board:       "a",
				LastUpdated: 3,
			},
			{
				ID:          3,
				PostCtr:     1,
				Board:       "c",
				LastUpdated: 4,
			},
		},
	}

	board, err := GetAllBoard()
	if err != nil {
		t.Fatal(err)
	}
	assertDeepEquals(t, board, &std)
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
						ID:          3,
						PostCtr:     1,
						Board:       "c",
						LastUpdated: 4,
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
			assertDeepEquals(t, board, &c.std)
		})
	}
}

func testGetThread(t *testing.T) {
	t.Parallel()

	thread1 := types.Thread{
		PostCtr: 3,
		Post: types.Post{
			ID: 1,
		},
		Board: "a",
		Posts: []types.Post{
			{
				ID:   2,
				Body: "foo",
			},
			{
				ID: 4,
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
					ID: 3,
				},
				Board: "c",
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
			assertDeepEquals(t, thread, c.std)
		})
	}
}
