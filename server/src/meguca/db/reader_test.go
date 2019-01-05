package db

import (
	"database/sql"
	"meguca/common"
	"meguca/config"
	"meguca/imager/assets"
	. "meguca/test"
	"reflect"
	"testing"
)

var sampleModerationEntry = common.ModerationEntry{
	Type:   common.BanPost,
	Length: 0,
	By:     "admin",
	Data:   "test",
}

func prepareThreads(t *testing.T) {
	t.Helper()
	assertTableClear(t, "boards", "images")

	boards := [...]BoardConfigs{
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
		err := InTransaction(false, func(tx *sql.Tx) error {
			return WriteBoard(tx, b)
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	threads := [...]Thread{
		{
			ID:        1,
			Board:     "a",
			ReplyTime: 1,
			BumpTime:  1,
		},
		{
			ID:        3,
			Board:     "c",
			ReplyTime: 3,
			BumpTime:  5,
		},
	}
	posts := [...]Post{
		{
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					ID:        1,
					Image:     &assets.StdJPEG,
					Moderated: true,
				},
				OP:    1,
				Board: "a",
			},
			Password: []byte("foo"),
			IP:       "::1",
		},
		{
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					ID: 3,
					Links: []common.Link{
						{
							ID:    1,
							OP:    1,
							Board: "a",
						},
					},
					Commands: []common.Command{
						{
							Type: common.Flip,
							Flip: true,
						},
					},
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
	err := InTransaction(false, func(tx *sql.Tx) (err error) {
		for i := len(threads); i < len(posts); i++ {
			if err = WritePost(tx, posts[i]); err != nil {
				return
			}
		}
		return
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = sq.Update("posts").Set("moderated", true).Where("id = 1").Exec()
	if err != nil {
		t.Fatal(err)
	}
	s := sampleModerationEntry
	_, err = sq.Insert("post_moderation").
		Columns("post_id", "type", "by", "length", "data").
		Values(1, s.Type, s.By, s.Length, s.Data).
		Exec()
	if err != nil {
		t.Fatal(err)
	}

}

func TestReader(t *testing.T) {
	prepareThreads(t)

	t.Run("GetAllBoard", testGetAllBoard)
	t.Run("GetBoard", testGetBoard)
	t.Run("GetPost", testGetPost)
	t.Run("GetThread", testGetThread)
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
			Links: []common.Link{
				{
					ID:    1,
					OP:    1,
					Board: "a",
				},
			},
			Commands: []common.Command{
				{
					Type: common.Flip,
					Flip: true,
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

	std := []common.Thread{
		{
			Post: common.Post{
				ID: 3,
				Links: []common.Link{
					{
						ID:    1,
						OP:    1,
						Board: "a",
					},
				},
				Commands: []common.Command{
					{
						Type: common.Flip,
						Flip: true,
					},
				},
			},
			PostCtr:   1,
			Board:     "c",
			ReplyTime: 3,
			BumpTime:  5,
		},
		{
			Post: common.Post{
				ID:         1,
				Image:      &assets.StdJPEG,
				Moderated:  true,
				Moderation: []common.ModerationEntry{sampleModerationEntry},
			},
			PostCtr:   3,
			ImageCtr:  1,
			Board:     "a",
			ReplyTime: 1,
			BumpTime:  1,
		},
	}

	board, err := GetAllBoardCatalog()
	if err != nil {
		t.Fatal(err)
	}
	for i := range board.Threads {
		assertImage(t, &board.Threads[i], std[i].Image)
	}
	AssertDeepEquals(t, board.Threads, std)
}

// Assert image equality and then override to not compare pointer addresses
// with reflection
func assertImage(t *testing.T, thread *common.Thread, std *common.Image) {
	t.Helper()
	if std != nil {
		if thread.Image == nil {
			t.Fatalf("no image on thread %d", thread.ID)
		}
		AssertDeepEquals(t, *thread.Image, *std)
		thread.Image = std
	}
}

func testGetBoard(t *testing.T) {
	t.Parallel()

	cases := [...]struct {
		name, id string
		std      []common.Thread
	}{
		{
			name: "full",
			id:   "c",
			std: []common.Thread{
				{
					Post: common.Post{
						ID: 3,
						Links: []common.Link{
							{
								ID:    1,
								OP:    1,
								Board: "a",
							},
						},
						Commands: []common.Command{
							{
								Type: common.Flip,
								Flip: true,
							},
						},
					},
					PostCtr:   1,
					Board:     "c",
					ReplyTime: 3,
					BumpTime:  5,
				},
			},
		},
		{
			name: "empty",
			id:   "z",
			std:  []common.Thread{},
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			board, err := GetBoardCatalog(c.id)
			if err != nil {
				t.Fatal(err)
			}
			for i := range board.Threads {
				assertImage(t, &board.Threads[i], c.std[i].Image)
			}
			AssertDeepEquals(t, board.Threads, c.std)
		})
	}
}

func testGetThread(t *testing.T) {
	t.Parallel()

	thread1 := common.Thread{
		PostCtr:   3,
		ImageCtr:  1,
		ReplyTime: 1,
		BumpTime:  1,
		Board:     "a",
		Post: common.Post{
			ID:         1,
			Image:      &assets.StdJPEG,
			Moderated:  true,
			Moderation: []common.ModerationEntry{sampleModerationEntry},
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
				Board:     "c",
				ReplyTime: 3,
				BumpTime:  5,
				PostCtr:   1,
				Post: common.Post{
					ID: 3,
					Links: []common.Link{
						{
							ID:    1,
							OP:    1,
							Board: "a",
						},
					},
					Commands: []common.Command{
						{
							Type: common.Flip,
							Flip: true,
						},
					},
				},
				Posts: []common.Post{},
			},
		},
		{
			name: "nonexistent thread",
			id:   99,
			err:  sql.ErrNoRows,
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
			assertImage(t, &thread, c.std.Image)
			AssertDeepEquals(t, thread, c.std)
		})
	}
}
