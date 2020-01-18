package db

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/test"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
)

// Insert sample thread and return its ID
func insertSampleThread(t *testing.T) (id uint64, authKey auth.AuthKey) {
	t.Helper()

	authKey = genToken(t)
	id, err := InsertThread(context.Background(), ThreadInsertParams{
		Subject: "test",
		Tags:    []string{"animu", "mango"},
		PostInsertParamsCommon: PostInsertParamsCommon{
			AuthKey: &authKey,
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

		authKey := genToken(t)
		err = InTransaction(ctx, func(tx pgx.Tx) (err error) {
			replies[i], err = InsertPost(ctx, tx, ReplyInsertParams{
				Thread: threads[i],
				PostInsertParamsCommon: PostInsertParamsCommon{
					AuthKey: &authKey,
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
	test.AssertJSON(t, bytes.NewReader(buf), map[uint64]interface{}{
		threads[0]: map[string]interface{}{
			"open_posts": map[uint64]interface{}{
				replies[0]: map[string]interface{}{
					"body":            nil,
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
		threads[1]: map[string]interface{}{
			"open_posts": map[uint64]interface{}{
				replies[1]: map[string]interface{}{
					"body":            nil,
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
