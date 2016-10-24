package db

import (
	"fmt"
	"testing"
	"time"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	. "github.com/bakape/meguca/test"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
)

func TestExpireUserSessions(t *testing.T) {
	assertTableClear(t, "accounts")

	expired := time.Now().Add(-time.Hour)
	samples := []auth.User{
		{
			ID: "1",
			Sessions: []auth.Session{
				{
					Token:   "foo",
					Expires: expired,
				},
				{
					Token:   "bar",
					Expires: time.Now().Add(time.Hour),
				},
			},
		},
		{
			ID: "2",
			Sessions: []auth.Session{
				{
					Token:   "baz",
					Expires: expired,
				},
			},
		},
	}
	assertInsert(t, "accounts", samples)

	if err := expireUserSessions(); err != nil {
		t.Fatal(err)
	}

	t.Run("not expired", func(t *testing.T) {
		t.Parallel()
		var res []auth.Session
		if err := All(GetAccount("1").Field("sessions"), &res); err != nil {
			t.Fatal(err)
		}
		if len(res) != 1 {
			t.Errorf("unexpected session count: %d", len(res))
		}
		token := res[0].Token
		if token != "bar" {
			t.Errorf("unexpected session token: %s", token)
		}
	})

	t.Run("expired", func(t *testing.T) {
		t.Parallel()
		var res []auth.Session
		if err := All(GetAccount("2").Field("sessions"), &res); err != nil {
			t.Fatal(err)
		}
		if res != nil {
			t.Fatal("session not cleared")
		}
	})
}

func TestOpenPostClosing(t *testing.T) {
	assertTableClear(t, "posts")

	tooOld := time.Now().Add(-time.Minute * 31).Unix()
	log := [][]byte{[]byte{1, 2, 3}}
	posts := []types.DatabasePost{
		{
			StandalonePost: types.StandalonePost{
				Post: types.Post{
					ID:      1,
					Editing: true,
					Time:    tooOld,
				},
			},
			Log: log,
		},
		{
			StandalonePost: types.StandalonePost{
				Post: types.Post{
					ID:      2,
					Editing: true,
					Time:    time.Now().Unix(),
				},
			},
		},
	}
	assertInsert(t, "posts", posts)

	if err := closeDanglingPosts(); err != nil {
		t.Fatal(err)
	}

	cases := [...]struct {
		name    string
		id      int64
		editing bool
	}{
		{"closed", 1, false},
		{"untouched", 2, true},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			var editing bool
			err := One(FindPost(c.id).Field("editing"), &editing)
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
		var isAppended bool
		q := FindPost(1).Field("log").Nth(-1).Eq([]byte("061"))
		if err := One(q, &isAppended); err != nil {
			t.Fatal(err)
		}
		if !isAppended {
			t.Error("log not updated")
		}
	})

	t.Run("lastUpdated field", func(t *testing.T) {
		t.Parallel()
		var lu int64
		q := FindPost(1).Field("lastUpdated")
		if err := One(q, &lu); err != nil {
			t.Fatal(err)
		}
		if lu <= tooOld || lu > time.Now().Unix() {
			t.Fatalf("unexpected lastUpdated time: %d", lu)
		}
	})
}

func TestImageTokenExpiry(t *testing.T) {
	assertTableClear(t, "images")

	const SHA1 = "123"
	assertInsert(t, "images", types.ProtoImage{
		ImageCommon: types.ImageCommon{
			SHA1:     "123",
			FileType: types.JPEG,
		},
		Posts: 7,
	})

	expired := time.Now().Add(-time.Minute)
	tokens := [...]allocationToken{
		{
			SHA1:    SHA1,
			Expires: expired,
		},
		{
			SHA1:    SHA1,
			Expires: expired,
		},
		{
			SHA1:    SHA1,
			Expires: time.Now().Add(time.Minute),
		},
	}
	assertInsert(t, "imageTokens", tokens)

	if err := expireImageTokens(); err != nil {
		t.Fatal(err)
	}

	var posts int
	if err := One(GetImage(SHA1).Field("posts"), &posts); err != nil {
		t.Fatal(err)
	}
	if posts != 5 {
		t.Errorf("unexpected reference count: %d", posts)
	}
}

func TestDeleteThread(t *testing.T) {
	assertTableClear(t, "threads", "posts", "images")

	t.Run("without images", deleteThreadWithoutImages)
	t.Run("with images", deleteThreadWithImages)
	t.Run("nonexitant thread", deleteMissingThread)
}

func deleteThreadWithoutImages(t *testing.T) {
	t.Parallel()

	assertInsert(t, "threads", types.DatabaseThread{
		ID: 1,
	})

	posts := [...]types.DatabasePost{
		{
			StandalonePost: types.StandalonePost{
				Post: types.Post{
					ID: 1,
				},
				OP: 1,
			},
		},
		{
			StandalonePost: types.StandalonePost{
				Post: types.Post{
					ID: 2,
				},
				OP: 1,
			},
		},
	}
	assertInsert(t, "posts", posts)

	if err := DeleteThread(1); err != nil {
		t.Fatal(err)
	}

	t.Run("thread", func(t *testing.T) {
		t.Parallel()
		assertDeleted(t, FindThread(1), true)
	})

	for i := int64(1); i <= 2; i++ {
		id := i
		t.Run(fmt.Sprintf("post %d", id), func(t *testing.T) {
			t.Parallel()
			assertDeleted(t, FindPost(id), true)
		})
	}
}

func assertDeleted(t *testing.T, q r.Term, del bool) {
	var deleted bool
	if err := One(q.Eq(nil), &deleted); err != nil {
		t.Fatal(err)
	}
	if deleted != del {
		LogUnexpected(t, del, deleted)
	}
}

func deleteMissingThread(t *testing.T) {
	t.Parallel()
	if err := DeleteThread(99); err != nil {
		t.Fatal(err)
	}
}

func deleteThreadWithImages(t *testing.T) {
	t.Parallel()

	images := [...]types.ProtoImage{
		{
			ImageCommon: types.ImageCommon{
				SHA1: "111",
			},
			Posts: 7,
		},
		{
			ImageCommon: types.ImageCommon{
				SHA1: "122",
			},
			Posts: 8,
		},
	}
	assertInsert(t, "images", images)

	assertInsert(t, "threads", types.DatabaseThread{
		ID: 11,
	})

	posts := [...]types.DatabasePost{
		{
			StandalonePost: types.StandalonePost{
				OP: 11,
				Post: types.Post{
					ID: 11,
					Image: &types.Image{
						ImageCommon: types.ImageCommon{
							SHA1: "111",
						},
					},
				},
			},
		},
		{
			StandalonePost: types.StandalonePost{
				OP: 11,
				Post: types.Post{
					ID: 12,
					Image: &types.Image{
						ImageCommon: types.ImageCommon{
							SHA1: "122",
						},
					},
				},
			},
		},
	}
	assertInsert(t, "posts", posts)

	if err := DeleteThread(11); err != nil {
		t.Fatal(err)
	}

	t.Run("thread", func(t *testing.T) {
		t.Parallel()
		assertDeleted(t, FindThread(11), true)
	})

	cases := [...]struct {
		id       int64
		sha1     string
		refCount int
	}{
		{11, "111", 6},
		{12, "122", 7},
	}

	for i := range cases {
		c := cases[i]
		t.Run(fmt.Sprintf("post %d", c.id), func(t *testing.T) {
			t.Parallel()
			assertDeleted(t, FindPost(c.id), true)
		})
		t.Run("image ref count "+c.sha1, func(t *testing.T) {
			t.Parallel()
			assertImageRefCount(t, c.sha1, c.refCount)
		})
	}
}

func TestDeleteUnusedBoards(t *testing.T) {
	assertTableClear(t, "boards", "threads", "posts")

	t.Run("no unused boards", func(t *testing.T) {
		(*config.Get()).PruneBoards = true
		if err := deleteUnusedBoards(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("board with no threads", func(t *testing.T) {
		(*config.Get()).PruneBoards = true
		assertInsert(t, "boards", config.DatabaseBoardConfigs{
			Created: time.Now().Add((-week - 1) * time.Second),
			BoardConfigs: config.BoardConfigs{
				ID: "l",
			},
		})

		if err := deleteUnusedBoards(); err != nil {
			t.Fatal(err)
		}
		assertDeleted(t, r.Table("boards").Get("l"), true)
	})

	t.Run("pruning disabled", func(t *testing.T) {
		(*config.Get()).PruneBoards = false
		assertInsert(t, "boards", config.DatabaseBoardConfigs{
			Created: time.Now().Add((-week - 1) * time.Second),
			BoardConfigs: config.BoardConfigs{
				ID: "x",
			},
		})

		if err := deleteUnusedBoards(); err != nil {
			t.Fatal(err)
		}
		assertDeleted(t, r.Table("boards").Get("x"), false)
	})

	t.Run("board with threads", testDeleteUnsusedBoards)
}

func testDeleteUnsusedBoards(t *testing.T) {
	(*config.Get()).PruneBoards = true
	expired := time.Now().Add((-week - 1) * time.Second)
	fresh := time.Now()

	boards := [...]config.DatabaseBoardConfigs{
		{
			Created: expired,
			BoardConfigs: config.BoardConfigs{
				ID: "a",
			},
		},
		{
			Created: expired,
			BoardConfigs: config.BoardConfigs{
				ID: "c",
			},
		},
	}
	assertInsert(t, "boards", boards)

	threads := [...]types.DatabaseThread{
		{
			ID:    1,
			Board: "a",
		},
		{
			ID:    3,
			Board: "c",
		},
	}
	assertInsert(t, "threads", threads)

	posts := [...]types.DatabasePost{
		{
			StandalonePost: types.StandalonePost{
				Post: types.Post{
					ID:   1,
					Time: expired.Unix(),
				},
				OP:    1,
				Board: "a",
			},
		},
		{
			StandalonePost: types.StandalonePost{
				Post: types.Post{
					ID:   3,
					Time: expired.Unix(),
				},
				OP:    3,
				Board: "c",
			},
		},
		{
			StandalonePost: types.StandalonePost{
				Post: types.Post{
					ID:   4,
					Time: fresh.Unix(),
				},
				OP:    3,
				Board: "c",
			},
		},
	}
	assertInsert(t, "posts", posts)

	if err := deleteUnusedBoards(); err != nil {
		t.Fatal(err)
	}

	cases := [...]struct {
		name    string
		deleted bool
		board   string
		id      int64
	}{
		{"deleted", true, "a", 1},
		{"untouched", false, "c", 3},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			t.Run("board", func(t *testing.T) {
				t.Parallel()
				assertDeleted(t, r.Table("boards").Get(c.board), c.deleted)
			})
			t.Run("thread", func(t *testing.T) {
				t.Parallel()
				assertDeleted(t, FindThread(c.id), c.deleted)
			})
			t.Run("post", func(t *testing.T) {
				t.Parallel()
				assertDeleted(t, FindPost(c.id), c.deleted)
			})
		})
	}
}
