package db

import (
	"testing"
	"time"

	"github.com/bakape/meguca/common"
	. "github.com/bakape/meguca/test"
	"github.com/lib/pq"
)

// const eightDays = time.Hour * 24 * 8

func TestOpenPostClosing(t *testing.T) {
	assertTableClear(t, "boards")
	writeSampleBoard(t)
	writeSampleThread(t)

	tooOld := time.Now().Add(-time.Minute * 31).Unix()
	posts := [...]DatabasePost{
		{
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					ID:      2,
					Editing: true,
					Time:    tooOld,
				},
				OP: 1,
			},
		},
		{
			StandalonePost: common.StandalonePost{
				Post: common.Post{
					ID:      3,
					Editing: true,
					Time:    time.Now().Unix(),
				},
				OP: 1,
			},
		},
	}
	for _, p := range posts {
		if err := WritePost(nil, p); err != nil {
			t.Fatal(err)
		}
	}

	if err := closeDanglingPosts(); err != nil {
		t.Fatal(err)
	}

	cases := [...]struct {
		name    string
		id      uint64
		editing bool
	}{
		{"closed", 2, false},
		{"untouched", 3, true},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			var editing bool
			err := db.
				QueryRow(`SELECT editing FROM posts WHERE id = $1`, c.id).
				Scan(&editing)
			if err != nil {
				t.Fatal(err)
			}
			if editing != c.editing {
				LogUnexpected(t, c.editing, editing)
			}
		})
	}

	t.Run("log update", func(t *testing.T) {
		t.Parallel()
		assertLogContains(t, 1, "062")
	})
}

func assertLogContains(t *testing.T, id uint64, msgs ...string) {
	var contains bool
	err := db.
		QueryRow(
			`SELECT true FROM threads WHERE id = $1 and log @> $2`,
			id, pq.StringArray(msgs),
		).
		Scan(&contains)
	if err != nil {
		t.Fatal(err)
	}
	if !contains {
		t.Errorf("replication log does not contain %v", msgs)
	}
}

// func TestImageTokenExpiry(t *testing.T) {
// 	assertTableClear(t, "images")

// 	const SHA1 = "123"
// 	assertInsert(t, "images", common.ProtoImage{
// 		ImageCommon: common.ImageCommon{
// 			SHA1:     "123",
// 			FileType: common.JPEG,
// 		},
// 		Posts: 7,
// 	})

// 	expired := time.Now().Add(-time.Minute)
// 	tokens := [...]allocationToken{
// 		{
// 			SHA1:    SHA1,
// 			Expires: expired,
// 		},
// 		{
// 			SHA1:    SHA1,
// 			Expires: expired,
// 		},
// 		{
// 			SHA1:    SHA1,
// 			Expires: time.Now().Add(time.Minute),
// 		},
// 	}
// 	assertInsert(t, "imageTokens", tokens)

// 	if err := expireImageTokens(); err != nil {
// 		t.Fatal(err)
// 	}

// 	var posts int
// 	if err := One(GetImage(SHA1).Field("posts"), &posts); err != nil {
// 		t.Fatal(err)
// 	}
// 	if posts != 5 {
// 		t.Errorf("unexpected reference count: %d", posts)
// 	}
// }

// func TestDeleteThread(t *testing.T) {
// 	assertTableClear(t, "threads", "posts", "images")

// 	t.Run("without images", deleteThreadWithoutImages)
// 	t.Run("with images", deleteThreadWithImages)
// 	t.Run("nonexistent thread", deleteMissingThread)
// }

// func deleteThreadWithoutImages(t *testing.T) {
// 	t.Parallel()

// 	assertInsert(t, "threads", common.DatabaseThread{
// 		ID: 1,
// 	})

// 	posts := [...]common.DatabasePost{
// 		{
// 			StandalonePost: common.StandalonePost{
// 				Post: common.Post{
// 					ID: 1,
// 				},
// 				OP: 1,
// 			},
// 		},
// 		{
// 			StandalonePost: common.StandalonePost{
// 				Post: common.Post{
// 					ID: 2,
// 				},
// 				OP: 1,
// 			},
// 		},
// 	}
// 	assertInsert(t, "posts", posts)

// 	if err := DeleteThread(1); err != nil {
// 		t.Fatal(err)
// 	}

// 	t.Run("thread", func(t *testing.T) {
// 		t.Parallel()
// 		assertDeleted(t, FindThread(1), true)
// 	})

// 	for i := uint64(1); i <= 2; i++ {
// 		id := i
// 		t.Run(fmt.Sprintf("post %d", id), func(t *testing.T) {
// 			t.Parallel()
// 			assertDeleted(t, FindPost(id), true)
// 		})
// 	}
// }

// func assertDeleted(t *testing.T, q r.Term, del bool) {
// 	var deleted bool
// 	if err := One(q.Eq(nil), &deleted); err != nil {
// 		t.Fatal(err)
// 	}
// 	if deleted != del {
// 		LogUnexpected(t, del, deleted)
// 	}
// }

// func deleteMissingThread(t *testing.T) {
// 	t.Parallel()
// 	if err := DeleteThread(99); err != nil {
// 		t.Fatal(err)
// 	}
// }

// func deleteThreadWithImages(t *testing.T) {
// 	t.Parallel()

// 	images := [...]common.ProtoImage{
// 		{
// 			ImageCommon: common.ImageCommon{
// 				SHA1: "111",
// 			},
// 			Posts: 7,
// 		},
// 		{
// 			ImageCommon: common.ImageCommon{
// 				SHA1: "122",
// 			},
// 			Posts: 8,
// 		},
// 	}
// 	assertInsert(t, "images", images)

// 	assertInsert(t, "threads", common.DatabaseThread{
// 		ID: 11,
// 	})

// 	posts := [...]common.DatabasePost{
// 		{
// 			StandalonePost: common.StandalonePost{
// 				OP: 11,
// 				Post: common.Post{
// 					ID: 11,
// 					Image: &common.Image{
// 						ImageCommon: common.ImageCommon{
// 							SHA1: "111",
// 						},
// 					},
// 				},
// 			},
// 		},
// 		{
// 			StandalonePost: common.StandalonePost{
// 				OP: 11,
// 				Post: common.Post{
// 					ID: 12,
// 					Image: &common.Image{
// 						ImageCommon: common.ImageCommon{
// 							SHA1: "122",
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}
// 	assertInsert(t, "posts", posts)

// 	if err := DeleteThread(11); err != nil {
// 		t.Fatal(err)
// 	}

// 	t.Run("thread", func(t *testing.T) {
// 		t.Parallel()
// 		assertDeleted(t, FindThread(11), true)
// 	})

// 	cases := [...]struct {
// 		id       uint64
// 		sha1     string
// 		refCount int
// 	}{
// 		{11, "111", 6},
// 		{12, "122", 7},
// 	}

// 	for i := range cases {
// 		c := cases[i]
// 		t.Run(fmt.Sprintf("post %d", c.id), func(t *testing.T) {
// 			t.Parallel()
// 			assertDeleted(t, FindPost(c.id), true)
// 		})
// 		t.Run("image ref count "+c.sha1, func(t *testing.T) {
// 			t.Parallel()
// 			assertImageRefCount(t, c.sha1, c.refCount)
// 		})
// 	}
// }

// func TestDeleteUnusedBoards(t *testing.T) {
// 	assertTableClear(t, "boards", "threads", "posts")
// 	config.Set(config.Configs{
// 		BoardExpiry: 7,
// 		PruneBoards: true,
// 	})

// 	t.Run("no unused boards", func(t *testing.T) {
// 		t.Parallel()

// 		if err := deleteUnusedBoards(); err != nil {
// 			t.Fatal(err)
// 		}
// 	})

// 	t.Run("board with no threads", func(t *testing.T) {
// 		t.Parallel()

// 		assertInsert(t, "boards", config.DatabaseBoardConfigs{
// 			Created: time.Now().Add(-eightDays),
// 			BoardConfigs: config.BoardConfigs{
// 				ID: "l",
// 			},
// 		})

// 		if err := deleteUnusedBoards(); err != nil {
// 			t.Fatal(err)
// 		}
// 		assertDeleted(t, r.Table("boards").Get("l"), true)
// 	})

// 	t.Run("pruning disabled", func(t *testing.T) {
// 		(*config.Get()).PruneBoards = false
// 		assertInsert(t, "boards", config.DatabaseBoardConfigs{
// 			Created: time.Now().Add(-eightDays),
// 			BoardConfigs: config.BoardConfigs{
// 				ID: "x",
// 			},
// 		})

// 		if err := deleteUnusedBoards(); err != nil {
// 			t.Fatal(err)
// 		}
// 		assertDeleted(t, r.Table("boards").Get("x"), false)
// 	})

// 	t.Run("board with threads", testDeleteUnusedBoards)
// }

// func testDeleteUnusedBoards(t *testing.T) {
// 	config.Set(config.Configs{
// 		PruneBoards: true,
// 		BoardExpiry: 7,
// 	})
// 	expired := time.Now().Add(-eightDays)
// 	fresh := time.Now()

// 	boards := [...]config.DatabaseBoardConfigs{
// 		{
// 			Created: expired,
// 			BoardConfigs: config.BoardConfigs{
// 				ID: "a",
// 			},
// 		},
// 		{
// 			Created: expired,
// 			BoardConfigs: config.BoardConfigs{
// 				ID: "c",
// 			},
// 		},
// 	}
// 	assertInsert(t, "boards", boards)

// 	threads := [...]common.DatabaseThread{
// 		{
// 			ID:    1,
// 			Board: "a",
// 		},
// 		{
// 			ID:    3,
// 			Board: "c",
// 		},
// 	}
// 	assertInsert(t, "threads", threads)

// 	posts := [...]common.DatabasePost{
// 		{
// 			StandalonePost: common.StandalonePost{
// 				Post: common.Post{
// 					ID:   1,
// 					Time: expired.Unix(),
// 				},
// 				OP:    1,
// 				Board: "a",
// 			},
// 		},
// 		{
// 			StandalonePost: common.StandalonePost{
// 				Post: common.Post{
// 					ID:   3,
// 					Time: expired.Unix(),
// 				},
// 				OP:    3,
// 				Board: "c",
// 			},
// 		},
// 		{
// 			StandalonePost: common.StandalonePost{
// 				Post: common.Post{
// 					ID:   4,
// 					Time: fresh.Unix(),
// 				},
// 				OP:    3,
// 				Board: "c",
// 			},
// 		},
// 	}
// 	assertInsert(t, "posts", posts)

// 	if err := deleteUnusedBoards(); err != nil {
// 		t.Fatal(err)
// 	}

// 	cases := [...]struct {
// 		name    string
// 		deleted bool
// 		board   string
// 		id      uint64
// 	}{
// 		{"deleted", true, "a", 1},
// 		{"untouched", false, "c", 3},
// 	}

// 	for i := range cases {
// 		c := cases[i]
// 		t.Run(c.name, func(t *testing.T) {
// 			t.Parallel()
// 			t.Run("board", func(t *testing.T) {
// 				t.Parallel()
// 				assertDeleted(t, r.Table("boards").Get(c.board), c.deleted)
// 			})
// 			t.Run("thread", func(t *testing.T) {
// 				t.Parallel()
// 				assertDeleted(t, FindThread(c.id), c.deleted)
// 			})
// 			t.Run("post", func(t *testing.T) {
// 				t.Parallel()
// 				assertDeleted(t, FindPost(c.id), c.deleted)
// 			})
// 		})
// 	}
// }

// func TestDeleteOldThreads(t *testing.T) {
// 	assertTableClear(t, "posts", "threads")
// 	config.Set(config.Configs{
// 		ThreadExpiry: 7,
// 	})

// 	t.Run("no expired threads", func(t *testing.T) {
// 		(*config.Get()).PruneThreads = true
// 		if err := deleteOldThreads(); err != nil {
// 			t.Fatal(err)
// 		}
// 	})

// 	assertInsert(t, "threads", []common.DatabaseThread{
// 		{ID: 1},
// 		{ID: 2},
// 	})
// 	assertInsert(t, "posts", []common.DatabasePost{
// 		{
// 			StandalonePost: common.StandalonePost{
// 				Post: common.Post{
// 					ID:   1,
// 					Time: time.Now().Add(-eightDays).Unix(),
// 				},
// 				OP: 1,
// 			},
// 		},
// 		{
// 			StandalonePost: common.StandalonePost{
// 				Post: common.Post{
// 					ID:   2,
// 					Time: time.Now().Unix(),
// 				},
// 				OP: 2,
// 			},
// 		},
// 	})

// 	t.Run("pruning disabled", func(t *testing.T) {
// 		(*config.Get()).PruneThreads = false
// 		if err := deleteOldThreads(); err != nil {
// 			t.Fatal(err)
// 		}
// 		assertDeleted(t, FindPost(1), false)
// 		assertDeleted(t, FindThread(1), false)
// 	})

// 	t.Run("deleted", func(t *testing.T) {
// 		(*config.Get()).PruneThreads = true
// 		if err := deleteOldThreads(); err != nil {
// 			t.Fatal(err)
// 		}
// 		for i := uint64(1); i <= 2; i++ {
// 			assertDeleted(t, FindPost(i), i == 1)
// 			assertDeleted(t, FindThread(i), i == 1)
// 		}
// 	})
// }
