package db

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"sort"
	"testing"
	"time"

	"github.com/bakape/meguca/test"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
)

// Insert sample thread and return its ID
func insertSampleThread(t *testing.T) (id uint64, pubKey uint64) {
	t.Helper()

	pubKey, _ = insertSamplePubKey(t)
	id, err := InsertThread(ThreadInsertParams{
		Subject: "test",
		Tags:    []string{"animu", "mango"},
		PostInsertParamsCommon: PostInsertParamsCommon{
			PublicKey: &pubKey,
			Body:      []byte("{}"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if id == 0 {
		t.Fatal("id not set")
	}
	return
}

func TestInsertThread(t *testing.T) {
	t.Parallel()

	id, _ := insertSampleThread(t)

	exists, err := ThreadExists(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	test.AssertEquals(t, exists, true)

	exists, err = ThreadExists(context.Background(), 456636351)
	if err != nil {
		t.Fatal(err)
	}
	test.AssertEquals(t, exists, false)
}

func TestGetFeedData(t *testing.T) {
	clearTables(t, "threads")

	var (
		threads, replies [2]uint64
		err              error
		ctx              = context.Background()

		// Postgres seems to have different timestamp rounding
		now = time.Now().Round(time.Second)
	)

	for i := range threads {
		threads[i], _ = insertSampleThread(t)
		_, err = db.Exec(
			ctx,
			`update posts
			set open = false,
				created_on = $2
			where id = $1`,
			threads[i],
			pgtype.Timestamptz{
				Time:   now.Add(-time.Hour),
				Status: pgtype.Present,
			},
		)
		if err != nil {
			t.Fatal(err)
		}

		pubKey, _ := insertSamplePubKey(t)
		err = InTransaction(ctx, func(tx pgx.Tx) (err error) {
			replies[i], _, err = InsertPost(tx, ReplyInsertParams{
				Thread: threads[i],
				PostInsertParamsCommon: PostInsertParamsCommon{
					PublicKey: &pubKey,
					Body:      []byte("{}"),
				},
			})
			return
		})
		if err != nil {
			t.Fatal(err)
		}

		// Correct to match timestamp
		_, err = db.Exec(
			ctx,
			`update posts
			set created_on = $2
			where id = $1`,
			replies[i],
			pgtype.Timestamptz{
				Time:   now,
				Status: pgtype.Present,
			},
		)
		if err != nil {
			t.Fatal(err)
		}
	}

	buf, err := GetFeedData()
	if err != nil {
		t.Fatal(err)
	}
	ut := now.Unix()
	test.AssertJSON(t, bytes.NewReader(buf), []map[string]interface{}{
		{
			"thread": threads[0],
			"open_posts": map[uint64]map[string]interface{}{
				replies[0]: {
					"body":            map[string]interface{}{},
					"thread":          threads[0],
					"has_image":       false,
					"created_on":      ut,
					"image_spoilered": false,
				},
			},
			"recent_posts": map[uint64]int64{
				replies[0]: ut,
			},
		},
		{
			"thread": threads[1],
			"open_posts": map[uint64]map[string]interface{}{
				replies[1]: {
					"body":            map[string]interface{}{},
					"thread":          threads[1],
					"has_image":       false,
					"created_on":      ut,
					"image_spoilered": false,
				},
			},
			"recent_posts": map[uint64]int64{
				replies[1]: ut,
			},
		},
	})
}

func TestReadThreads(t *testing.T) {
	clearTables(t, "threads")

	img, _, closeFiles := prepareSampleImage(t)
	closeFiles()
	thread, pubKey := insertSampleThread(t)
	thread2, _ := insertSampleThread(t)

	const imageName = "fuko_da.jpeg"
	// Postgres seems to have different timestamp rounding
	now := time.Now().Round(time.Second)
	unix := now.Unix()

	genPost := func(id, thread, page uint64) map[string]interface{} {
		return map[string]interface{}{
			"id":         id,
			"thread":     thread,
			"page":       page,
			"created_on": unix,
			"open":       true,
			"sage":       false,
			"name":       nil,
			"trip":       nil,
			"flag":       nil,
			"body":       map[string]interface{}{},
			"image":      nil,
		}
	}
	genThread := func(
		id, postCount, imageCount, lastPage uint64,
	) map[string]interface{} {
		return map[string]interface{}{
			"id":          id,
			"post_count":  postCount,
			"image_count": imageCount,
			"page":        0,
			"last_page":   lastPage,
			"created_on":  unix,
			"bumped_on":   unix,
			"subject":     "test",
			"tags":        []string{"animu", "mango"},
			"posts":       []map[string]interface{}{genPost(id, id, 0)},
		}
	}

	std := genThread(thread, 109, 1, 1)
	std["page"] = 0
	std["posts"] = []map[string]interface{}{
		{
			"id":         thread,
			"thread":     thread,
			"page":       0,
			"created_on": unix,
			"open":       true,
			"sage":       false,
			"name":       nil,
			"trip":       nil,
			"flag":       nil,
			"body":       map[string]interface{}{},
			"image": map[string]interface{}{
				"md5":          hex.EncodeToString(img.MD5[:]),
				"name":         imageName,
				"sha1":         hex.EncodeToString(img.SHA1[:]),
				"size":         1048576,
				"audio":        false,
				"title":        nil,
				"video":        false,
				"width":        300,
				"artist":       nil,
				"height":       300,
				"duration":     0,
				"file_type":    "JPEG",
				"spoilered":    false,
				"thumb_type":   "JPEG",
				"thumb_width":  150,
				"thumb_height": 150,
			},
		},
	}

	err := InTransaction(context.Background(), func(tx pgx.Tx) (err error) {
		_, err = InsertImage(
			context.Background(),
			tx,
			thread,
			pubKey,
			img.SHA1,
			imageName,
			false,
		)
		return
	})
	if err != nil {
		t.Fatal(err)
	}

	// Patch all timestamps to simplify comparison
	nowPgt := pgtype.Timestamptz{
		Time:   now,
		Status: pgtype.Present,
	}
	_, err = db.Exec(
		context.Background(),
		"update posts set created_on = $1",
		nowPgt,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(
		context.Background(),
		"update threads set bumped_on = $1",
		nowPgt,
	)
	if err != nil {
		t.Fatal(err)
	}

	err = InTransaction(context.Background(), func(tx pgx.Tx) (err error) {
		var id uint64
		for i := 1; i < 109; i++ {
			id, _, err = InsertPost(tx, ReplyInsertParams{
				Thread: thread,
				PostInsertParamsCommon: PostInsertParamsCommon{
					PublicKey: &pubKey,
					Body:      []byte("{}"),
				},
			})
			if err != nil {
				return
			}
			page := uint64(0)
			if i >= 100 {
				page = 1
			}
			std["posts"] = append(
				std["posts"].([]map[string]interface{}),
				genPost(id, thread, page),
			)
		}
		return
	})
	if err != nil {
		t.Fatal(err)
	}

	cloneStd := func() map[string]interface{} {
		re := make(map[string]interface{})
		for k, v := range std {
			re[k] = v
		}
		return re
	}

	firstPage := cloneStd()
	posts := std["posts"].([]map[string]interface{})
	firstPage["posts"] = posts[:100]

	lastPage := cloneStd()
	lastPage["page"] = 1
	lastPage["posts"] = append(
		[]map[string]interface{}{
			posts[0],
		},
		posts[100:]...,
	)

	last5 := cloneStd()
	last5["posts"] = append(
		[]map[string]interface{}{
			posts[0],
		},
		posts[len(posts)-5:]...,
	)

	cases := [...]struct {
		name string
		id   uint64
		page int
		std  map[string]interface{}
		err  error
	}{
		{
			name: "first page",
			id:   thread,
			std:  firstPage,
		},
		{
			name: "second page",
			id:   thread,
			page: 1,
			std:  lastPage,
		},
		{
			name: "last page",
			id:   thread,
			page: -1,
			std:  lastPage,
		},
		{
			name: "last 5 replies",
			id:   thread,
			page: -5,
			std:  last5,
		},
		{
			name: "no replies ;_;",
			id:   thread2,
			std:  genThread(thread2, 1, 0, 0),
		},
		{
			name: "nonexistent thread",
			id:   9999999,
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
				test.AssertJSON(t, bytes.NewReader(buf), c.std)
			}
		})
	}

	t.Run("get thread IDs", func(t *testing.T) {
		t.Parallel()

		ids, err := GetThreadIDs()
		if err != nil {
			t.Fatal(err)
		}
		sort.Sort(idSorter(ids))
		test.AssertEquals(t, ids, []uint64{thread, thread2})
	})

	t.Run("get page counts", func(t *testing.T) {
		t.Parallel()

		cases := [...]struct {
			name   string
			thread uint64
			last   int
		}{
			{
				name:   "small",
				thread: thread2,
				last:   0,
			},
			{
				name:   "bigger",
				thread: thread,
				last:   1,
			},
			{
				name:   "no thread",
				thread: thread2 + 20,
				last:   0,
			},
		}

		for i := range cases {
			c := cases[i]
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()

				last, err := GetLastPage(c.thread)
				if err != nil {
					t.Fatal(err)
				}
				test.AssertEquals(t, c.last, last)
			})
		}

	})

	t.Run("get tag list", func(t *testing.T) {
		t.Parallel()

		var res []string
		buf, err := GetTagList(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		err = json.Unmarshal(buf, &res)
		if err != nil {
			t.Fatal(err)
		}
		test.AssertEquals(t, res, []string{"animu", "mango"})
	})

	t.Run("get post parenthood", func(t *testing.T) {
		t.Parallel()

		threadRes, page, err := GetPostParenthood(posts[1]["id"].(uint64))
		if err != nil {
			t.Fatal(err)
		}
		test.AssertEquals(t, thread, threadRes)
		test.AssertEquals(t, page, uint32(0))
	})
}
