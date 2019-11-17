package db

import (
	"testing"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/imager/assets"
	"github.com/bakape/meguca/test"
	"github.com/jackc/pgx"
)

var sampleModerationEntry = common.ModerationEntry{
	Type:   common.BanPost,
	Length: 0,
	By:     "admin",
	Data:   "test",
}

func prepareThread(t *testing.T) (op Post, replies []Post) {
	t.Helper()
	assertTableClear(t, "images")

	err := WriteImage(assets.StdJPEG.ImageCommon)
	if err != nil {
		t.Fatal(err)
	}

	err = InTransaction(func(tx *pgx.Tx) (err error) {
		err = InsertThread(tx, "test", &op)
		if err != nil {
			return
		}

		var p Post
		for i := 0; i < 110; i++ {
			p = Post{
				StandalonePost: common.StandalonePost{
					OP: op.ID,
				},
			}
			err = InsertPost(tx, &p)
			if err != nil {
				return
			}
			replies = append(replies, p)
		}

		return
	})
	if err != nil {
		t.Fatal(err)
	}

	return
}

func TestReader(t *testing.T) {
	prepareThread(t)

	// t.Run("GetAllCatalog", func(t *testing.T) {
	// 	t.Parallel()

	// 	std := map[uint64]common.Thread{
	// 		3: {
	// 			PostCount:  1,
	// 			Board:      "c",
	// 			UpdateTime: 3,
	// 			BumpTime:   5,
	// 			ID:         3,
	// 			Posts: []common.Post{
	// 				{
	// 					ID: 3,
	// 					Links: map[uint64]common.Link{
	// 						1: {
	// 							OP:    1,
	// 							Board: "a",
	// 						},
	// 					},
	// 					Commands: []common.Command{
	// 						{
	// 							Type: common.Flip,
	// 							Flip: true,
	// 						},
	// 					},
	// 				},
	// 			},
	// 		},
	// 		1: {
	// 			ID:         1,
	// 			PostCount:  109,
	// 			ImageCount: 1,
	// 			Board:      "a",
	// 			UpdateTime: 1,
	// 			BumpTime:   1,
	// 			Posts: []common.Post{
	// 				{
	// 					ID:         1,
	// 					Image:      &assets.StdJPEG,
	// 					Moderation: []common.ModerationEntry{sampleModerationEntry},
	// 				},
	// 			},
	// 		},
	// 	}

	// 	buf, err := GetAllBoardCatalog()
	// 	if err != nil {
	// 		t.Fatal(err)
	// 	}
	// 	var catalog []common.Thread
	// 	test.DecodeJSON(t, buf, &catalog)
	// 	for i := range catalog {
	// 		thread := &catalog[i]
	// 		std := std[thread.ID]
	// 		t.Run("assert thread equality", func(t *testing.T) {
	// 			t.Parallel()

	// 			assertImage(t, thread, std.Posts[0].Image)
	// 			syncThreadVariables(thread, std)
	// 			test.AssertEquals(t, thread, &std)
	// 		})
	// 	}
	// })

	t.Run("GetPost", func(t *testing.T) {
		t.Parallel()
		// Does not exist
		_, err := GetPost(9999)
		if err != pgx.ErrNoRows {
			test.UnexpectedError(t, err)
		}

		// Valid read
		std := common.StandalonePost{
			Post: common.Post{
				ID: 3,
			},
			OP: 3,
		}
		buf, err := GetPost(3)
		if err != nil {
			t.Fatal(err)
		}
		var p common.StandalonePost
		test.DecodeJSON(t, buf, &p)
		test.AssertEquals(t, p, std)
	})

	t.Run("GetThread", func(t *testing.T) {
		t.Parallel()

		thread1 := common.Thread{
			PostCount:  109,
			ImageCount: 1,
			UpdateTime: 1,
			BumpTime:   1,
			Board:      "a",
			ID:         1,
			Posts: []common.Post{
				{
					ID:         1,
					Image:      &assets.StdJPEG,
					Moderation: []common.ModerationEntry{sampleModerationEntry},
				},
				{
					ID:   2,
					Body: "foo",
				},
			},
		}
		for i := uint64(4); i <= 110; i++ {
			thread1.Posts = append(thread1.Posts, common.Post{
				ID:   i,
				Page: (uint32(i) - 1) / 100,
			})
		}

		firstPage := thread1
		firstPage.Posts = firstPage.Posts[:99]

		last5 := thread1
		last5.Posts = append(
			[]common.Post{thread1.Posts[0]},
			last5.Posts[len(thread1.Posts)-5:]...,
		)

		lastPage := thread1
		lastPage.Page = 1
		lastPage.Posts = append(
			[]common.Post{thread1.Posts[0]},
			lastPage.Posts[99:]...,
		)

		cases := [...]struct {
			name string
			id   uint64
			page int
			std  common.Thread
			err  error
		}{
			{
				name: "first page",
				id:   1,
				std:  firstPage,
			},
			{
				name: "second page",
				id:   1,
				page: 1,
				std:  lastPage,
			},
			{
				name: "last page",
				id:   1,
				page: -1,
				std:  lastPage,
			},
			{
				name: "last 5 replies",
				id:   1,
				page: -5,
				std:  last5,
			},
			{
				name: "no replies ;_;",
				id:   3,
				std: common.Thread{
					Board:      "c",
					UpdateTime: 3,
					BumpTime:   5,
					PostCount:  1,
					ID:         3,
					Posts: []common.Post{
						{
							ID: 3,
						},
					},
				},
			},
			{
				name: "nonexistent thread",
				id:   99,
				err:  pgx.ErrNoRows,
			},
		}

		for i := range cases {
			c := cases[i]
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()

				buf, err := GetThread(c.id, c.page)
				if err != c.err {
					test.UnexpectedError(t, err)
				}
				if c.err == nil {
					var thread common.Thread
					test.DecodeJSON(t, buf, &thread)
					assertImage(t, &thread, c.std.Posts[0].Image)
					syncThreadVariables(&thread, c.std)
					test.AssertEquals(t, thread, c.std)
				}
			})
		}
	})

	// t.Run("GetPostCloseData", func(t *testing.T) {
	// 	t.Parallel()

	// 	res, err := GetPostCloseData(3)
	// 	if err != nil {
	// 		t.Fatal(err)
	// 	}

	// 	test.AssertJSON(t, bytes.NewReader(res.Commands), []common.Command{
	// 		{
	// 			Type: common.Flip,
	// 			Flip: true,
	// 		},
	// 	})
	// 	test.AssertEquals(
	// 		t,
	// 		res,
	// 		CloseData{

	// 		},
	// 	)
	// })
}

// Assert image equality and then override to not compare pointer addresses
// with reflection
func assertImage(t *testing.T, thread *common.Thread, std *common.Image) {
	t.Helper()
	if std != nil {
		if len(thread.Posts) == 0 || thread.Posts[0].Image == nil {
			t.Fatalf("no image on thread %d", thread.ID)
		}
		test.AssertEquals(t, *thread.Posts[0].Image, *std)
		thread.Posts[0].Image = std
	}
}

// Sync variables that are generated from external state and can not be easily
// tested
func syncThreadVariables(dst *common.Thread, src common.Thread) {
	dst.ID = src.ID
	dst.UpdateTime = src.UpdateTime
	dst.BumpTime = src.BumpTime
}

func TestOpenPostMetaFromPost(t *testing.T) {
	t.Parallel()

	test.AssertEquals(
		t,
		OpenPostMetaFromPost(
			common.Post{
				Page: 1,
				Body: "foo",
				Image: &common.Image{
					Spoiler: true,
				},
			},
		),
		OpenPostMeta{
			Page:      1,
			Body:      "foo",
			HasImage:  true,
			Spoilered: true,
		},
	)
}
