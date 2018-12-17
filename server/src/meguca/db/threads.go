package db

import (
	"database/sql"
	"errors"
	"fmt"
	"meguca/common"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/squirrel"
)

var (
	postCountCache           = make(map[uint64]uint64)
	postCountCacheMu         sync.RWMutex
	errTooManyWatchedThreads = common.StatusError{
		errors.New("too many watched threads"), 400}
)

// Diff of passed and actual thread posts counts
type ThreadPostCountDiff struct {
	Changed map[uint64]int `json:"changed"`
	Deleted []uint64       `json:"deleted"`
}

// Return diff of passed and actual thread post counts
func DiffThreadPostCounts(old map[uint64]uint64) (
	diff ThreadPostCountDiff, err error,
) {
	if len(old) > 1000 {
		err = errTooManyWatchedThreads
		return
	}

	postCountCacheMu.RLock()
	defer postCountCacheMu.RUnlock()

	diff.Changed = make(map[uint64]int, len(old))
	diff.Deleted = make([]uint64, 0)
	for thread, count := range old {
		actual, ok := postCountCache[thread]
		if !ok {
			diff.Deleted = append(diff.Deleted, thread)
		} else if delta := int(actual) - int(count); delta != 0 {
			diff.Changed[thread] = delta
		}
	}

	return
}

func loadThreadPostCounts() (err error) {
	r, err := sq.Select("op, count(*)").
		From("posts").
		GroupBy("op").
		Query()
	if err != nil {
		return
	}
	defer r.Close()

	postCountCacheMu.Lock()
	defer postCountCacheMu.Unlock()

	var thread, postCount uint64
	for r.Next() {
		err = r.Scan(&thread, &postCount)
		if err != nil {
			return
		}
		postCountCache[thread] = postCount
	}
	err = r.Err()
	if err != nil {
		return
	}

	return listenForThreadUpdates()
}

// Separate function for easier testing
func listenForThreadUpdates() (err error) {
	err = Listen("thread_deleted", func(msg string) (err error) {
		id, err := strconv.ParseUint(msg, 10, 64)
		if err != nil {
			return
		}

		postCountCacheMu.Lock()
		delete(postCountCache, id)
		postCountCacheMu.Unlock()
		return
	})
	if err != nil {
		return
	}

	return Listen("new_post_in_thread", func(msg string) (err error) {
		split := strings.Split(msg, ",")
		if len(split) != 2 {
			return fmt.Errorf("invalid message: `%s`", msg)
		}
		id, err := strconv.ParseUint(split[0], 10, 64)
		if err != nil {
			return
		}
		postCount, err := strconv.ParseUint(split[1], 10, 64)
		if err != nil {
			return
		}

		postCountCacheMu.Lock()
		postCountCache[id] = postCount
		postCountCacheMu.Unlock()
		return
	})
}

// Thread is a template for writing new threads to the database
type Thread struct {
	ID                  uint64
	PostCtr, ImageCtr   uint32
	ReplyTime, BumpTime int64
	Subject, Board      string
}

// ThreadCounter retrieves the progress counter of a thread
func ThreadCounter(id uint64) (uint64, error) {
	q := sq.Select("replyTime").
		From("threads").
		Where("id = ?", id)
	return getCounter(q)
}

// ValidateOP confirms the specified thread exists on specific board
func ValidateOP(id uint64, board string) (valid bool, err error) {
	err = sq.Select("true").
		From("threads").
		Where(squirrel.Eq{
			"id":    id,
			"board": board,
		}).
		QueryRow().
		Scan(&valid)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return
}

// InsertThread inserts a new thread into the database.
func InsertThread(tx *sql.Tx, subject string, p Post) (
	err error,
) {
	err = withTransaction(tx,
		sq.Insert("threads").
			Columns("board", "id", "replyTime", "bumpTime", "subject").
			Values(p.Board, p.ID, p.Time, p.Time, subject),
	).Exec()
	if err != nil {
		return
	}

	err = WritePost(tx, p, false, false)
	return
}

// WriteThread writes a thread and it's OP to the database. Only used for tests
// and migrations.
func WriteThread(tx *sql.Tx, t Thread, p Post) (err error) {
	passedTx := tx != nil
	if !passedTx {
		tx, err = db.Begin()
		if err != nil {
			return err
		}
	}

	err = withTransaction(tx, sq.
		Insert("threads").
		Columns("board", "id", "replyTime", "bumpTime", "subject").
		Values(
			t.Board,
			t.ID,
			t.ReplyTime,
			t.BumpTime,
			t.Subject,
		),
	).
		Exec()
	if err != nil {
		if !passedTx {
			tx.Rollback()
		}
		return err
	}

	err = WritePost(tx, p, false, false)
	if err != nil {
		if !passedTx {
			tx.Rollback()
		}
		return err
	}

	if !passedTx {
		return tx.Commit()
	}
	return nil
}

func queryThreadBool(id uint64, key string) (val bool, err error) {
	err = sq.Select(key).
		From("threads").
		Where("id = ?", id).
		QueryRow().
		Scan(&val)
	return
}

// CheckThreadLocked checks, if a thread has been locked by a moderator
func CheckThreadLocked(id uint64) (bool, error) {
	return queryThreadBool(id, "locked")
}

// Increment thread update, bump, post and image counters
func bumpThread(tx *sql.Tx, id uint64, bump bool) (err error) {
	now := time.Now().Unix()
	q := sq.Update("threads").Set("replyTime", now)

	if bump {
		var (
			r         rowScanner
			postCount uint
		)
		r, err = withTransaction(tx,
			sq.Select("count(*)").
				From("posts").
				Where("op = ?", id),
		).
			QueryRow()
		if err != nil {
			return
		}
		err = r.Scan(&postCount)
		if err != nil {
			return
		}

		if postCount < common.BumpLimit {
			q = q.Set("bumpTime", now)
		}
	}

	err = withTransaction(tx, q.Where("id = ?", id)).Exec()
	return err
}
