package db

import (
	"reflect"
	"testing"

	"github.com/bakape/meguca/common"
	. "github.com/bakape/meguca/test"
	r "github.com/dancannon/gorethink"
)

func TestReader(t *testing.T) {
	assertTableClear(t, "posts", "threads", "main")

	assertInsert(t, "posts", []common.DatabasePost{
		{
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					ID: 1,
				},
				OP:    1,
				Board: "a",
			},
			LastUpdated: 1,
			Log:         [][]byte{{1, 2, 3}},
		},
		{
			StandalonePost: common.StandalonePost{

				Post: common.Post{
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
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					ID: 4,
				},
				OP:    1,
				Board: "a",
			},
			LastUpdated: 3,
			Log:         [][]byte{{1}},
		},
		{
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					ID: 3,
				},
				OP:    3,
				Board: "c",
			},
			LastUpdated: 4,
			Log:         [][]byte{{1}, {2}},
		},
	})

	assertInsert(t, "threads", []common.DatabaseThread{
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

	assertInsert(t, "main", map[string]interface{}{
		"id":      "info",
		"postCtr": 3,
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
		UnexpectedError(t, err)
	}
	if !reflect.DeepEqual(post, common.StandalonePost{}) {
		t.Errorf("post not empty: %#v", post)
	}

	// Valid read
	std := common.StandalonePost{
		Post: common.Post{
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
	AssertDeepEquals(t, post, std)
}

func testGetAllBoard(t *testing.T) {
	t.Parallel()

	std := common.Board{
		{
			ID: 1,
			ThreadCommon: common.ThreadCommon{
				PostCtr:     3,
				Board:       "a",
				LastUpdated: 3,
			},
		},
		{
			ID: 3,
			ThreadCommon: common.ThreadCommon{
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
	AssertDeepEquals(t, board, std)
}

func testGetBoard(t *testing.T) {
	t.Parallel()

	cases := [...]struct {
		name, id string
		std      common.Board
	}{
		{
			name: "full",
			id:   "c",
			std: common.Board{
				{
					ID: 3,
					ThreadCommon: common.ThreadCommon{
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
			std:  common.Board{},
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
			AssertDeepEquals(t, board, c.std)
		})
	}
}

func testGetThread(t *testing.T) {
	t.Parallel()

	thread1 := common.Thread{
		ThreadCommon: common.ThreadCommon{
			PostCtr:     3,
			LastUpdated: 3,
			Board:       "a",
		},
		Post: common.Post{
			ID: 1,
		},
		Posts: []common.Post{
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
	sliced.Abbrev = true

	cases := [...]struct {
		name  string
		id    uint64
		lastN int
		std   common.Thread
		err   error
	}{
		{
			name: "full",
			id:   1,
			std:  thread1,
		},
		{
			name:  "last 1 reply",
			id:    1,
			lastN: 1,
			std:   sliced,
		},
		{
			name: "no replies ;_;",
			id:   3,
			std: common.Thread{
				ThreadCommon: common.ThreadCommon{
					PostCtr:     1,
					Board:       "c",
					LastUpdated: 4,
				},
				Post: common.Post{
					ID: 3,
				},
				Posts: []common.Post{},
			},
		},
		{
			name: "nonexistent thread",
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
				UnexpectedError(t, err)
			}
			AssertDeepEquals(t, thread, c.std)
		})
	}
}
