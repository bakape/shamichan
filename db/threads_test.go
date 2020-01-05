package db

import (
	"context"
	"testing"

	"github.com/bakape/meguca/test"
	"github.com/jackc/pgx/v4"
)

// Insert sample thread and return its ID
func insertSampleThread(t *testing.T) (id uint64) {
	t.Helper()

	id, err := InsertThread(context.Background(), ThreadInsertParams{
		Subject: "test",
		Tags:    []string{"animu", "mango"},
		PostInsertParamsCommon: PostInsertParamsCommon{
			AuthKey: genToken(t),
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
	id := insertSampleThread(t)

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
	)
	for i := range threads {
		threads[i] = insertSampleThread(t)
		_, err = db.Exec(
			context.Background(),
			`update posts
			set open = false,
				created_on = created_on - interval '1 hour'
			where id = $1`,
			threads[i],
		)
		if err != nil {
			t.Fatal(err)
		}

		err = InTransaction(context.Background(), func(tx pgx.Tx) (err error) {
			replies[i], err = InsertPost(context.Background(),
				tx, ReplyInsertParams{
					Thread: threads[i],
					PostInsertParamsCommon: PostInsertParamsCommon{
						AuthKey: genToken(t),
					},
				},
			)
			return
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	t.Fatal(`TODO: verify feed data`)
}
