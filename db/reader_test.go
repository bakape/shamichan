package db

import (
	"database/sql"
	"reflect"
	"testing"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/imager/assets"
	. "github.com/bakape/meguca/test"
)

func TestReader(t *testing.T) {
	assertTableClear(t, "boards", "images")

	boards := [...]config.DatabaseBoardConfigs{
		{
			BoardConfigs: config.BoardConfigs{
				ID:        "a",
				Eightball: []string{"yes"},
			},
		},
		{
			BoardConfigs: config.BoardConfigs{
				ID:        "c",
				Eightball: []string{"yes"},
			},
		},
	}
	for _, b := range boards {
		if err := WriteBoardConfigs(b, false); err != nil {
			t.Fatal(err)
		}
	}

	threads := [...]DatabaseThread{
		{
			ID:        1,
			Board:     "a",
			Log:       [][]byte{{1}},
			ReplyTime: 1,
			PostCtr:   3,
		},
		{
			ID:        3,
			Board:     "c",
			Log:       [][]byte{{1}},
			ReplyTime: 3,
			PostCtr:   1,
		},
	}
	links := common.LinkMap{
		1: {
			Board: "a",
			OP:    1,
		},
	}
	posts := [...]DatabasePost{
		{
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					ID:    1,
					Image: &assets.StdJPEG,
				},
				OP:    1,
				Board: "a",
			},
		},
		{
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					ID:    3,
					Links: links,
				},
				OP:    3,
				Board: "c",
			},
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
		},
		{
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					ID: 4,
				},
				OP:    1,
				Board: "a",
			},
		},
	}

	if err := WriteImage(assets.StdJPEG.ImageCommon); err != nil {
		t.Fatal(err)
	}
	for i := range threads {
		if err := WriteThread(threads[i], posts[i]); err != nil {
			t.Fatal(err)
		}
	}
	for i := len(threads); i < len(posts); i++ {
		if err := WritePost(nil, posts[i]); err != nil {
			t.Fatal(err)
		}
	}
	if err := WriteLinks(nil, 3, links); err != nil {
		t.Fatal(err)
	}

	t.Run("GetAllBoard", testGetAllBoard)
	t.Run("GetBoard", testGetBoard)
	t.Run("GetPost", testGetPost)
	// t.Run("GetThread", testGetThread)
}

func testGetPost(t *testing.T) {
	t.Parallel()

	// Does not exist
	post, err := GetPost(99)
	if err != sql.ErrNoRows {
		UnexpectedError(t, err)
	}
	if !reflect.DeepEqual(post, common.StandalonePost{}) {
		t.Errorf("post not empty: %#v", post)
	}

	// Valid read
	std := common.StandalonePost{
		Post: common.Post{
			ID: 3,
			Links: common.LinkMap{
				1: {
					Board: "a",
					OP:    1,
				},
			},
		},
		OP:    3,
		Board: "c",
	}
	post, err = GetPost(3)
	if err != nil {
		t.Fatal(err)
	}
	AssertDeepEquals(t, post, std)
}

func testGetAllBoard(t *testing.T) {
	t.Parallel()

	std := common.Board{
		{
			ID: 3,
			ThreadCommon: common.ThreadCommon{
				PostCtr:   1,
				Board:     "c",
				LogCtr:    1,
				ReplyTime: 3,
			},
		},
		{
			ID: 1,
			ThreadCommon: common.ThreadCommon{
				PostCtr:   3,
				Board:     "a",
				LogCtr:    1,
				ReplyTime: 1,
			},
			Image: &assets.StdJPEG,
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
						PostCtr:   1,
						Board:     "c",
						LogCtr:    1,
						ReplyTime: 3,
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

// func testGetThread(t *testing.T) {
// 	t.Parallel()

// 	thread1 := common.Thread{
// 		ThreadCommon: common.ThreadCommon{
// 			PostCtr:     3,
// 			LastUpdated: 3,
// 			Board:       "a",
// 		},
// 		Post: common.Post{
// 			ID: 1,
// 		},
// 		Posts: []common.Post{
// 			{
// 				ID:   2,
// 				Body: "foo",
// 			},
// 			{
// 				ID: 4,
// 			},
// 		},
// 	}
// 	sliced := thread1
// 	sliced.Posts = sliced.Posts[1:]
// 	sliced.Abbrev = true

// 	cases := [...]struct {
// 		name  string
// 		id    uint64
// 		lastN int
// 		std   common.Thread
// 		err   error
// 	}{
// 		{
// 			name: "full",
// 			id:   1,
// 			std:  thread1,
// 		},
// 		{
// 			name:  "last 1 reply",
// 			id:    1,
// 			lastN: 1,
// 			std:   sliced,
// 		},
// 		{
// 			name: "no replies ;_;",
// 			id:   3,
// 			std: common.Thread{
// 				ThreadCommon: common.ThreadCommon{
// 					PostCtr:     1,
// 					Board:       "c",
// 					LastUpdated: 4,
// 				},
// 				Post: common.Post{
// 					ID: 3,
// 				},
// 				Posts: []common.Post{},
// 			},
// 		},
// 		{
// 			name: "nonexistent thread",
// 			id:   99,
// 			err:  r.ErrEmptyResult,
// 		},
// 	}

// 	for i := range cases {
// 		c := cases[i]
// 		t.Run(c.name, func(t *testing.T) {
// 			t.Parallel()

// 			thread, err := GetThread(c.id, c.lastN)
// 			if err != c.err {
// 				UnexpectedError(t, err)
// 			}
// 			AssertDeepEquals(t, thread, c.std)
// 		})
// 	}
// }
