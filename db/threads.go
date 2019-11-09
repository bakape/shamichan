package db

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/Masterminds/squirrel"
	"github.com/bakape/meguca/common"
	"github.com/bakape/pg_util"
)

var (
	postCountCache           = make(map[uint64]uint64)
	postCountCacheMu         sync.RWMutex
	errTooManyWatchedThreads = common.StatusError{
		Err:  errors.New("too many watched threads"),
		Code: 400,
	}
)

// Diff of passed and actual thread posts counts
type ThreadPostCountDiff struct {
	Changed map[uint64]uint64 `json:"changed"`
	Deleted []uint64          `json:"deleted"`
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

	diff.Changed = make(map[uint64]uint64, len(old))
	diff.Deleted = make([]uint64, 0)
	for thread, count := range old {
		actual, ok := postCountCache[thread]
		if !ok {
			diff.Deleted = append(diff.Deleted, thread)
		} else if actual != count {
			diff.Changed[thread] = actual
		}
	}

	return
}

func loadThreadPostCounts() (err error) {
	err = readThreadPostCounts()
	if err != nil {
		return
	}
	return listenForThreadUpdates(nil)
}

func readThreadPostCounts() (err error) {
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
	return r.Err()
}

// Separate function for easier testing
func listenForThreadUpdates(canceller <-chan struct{}) (err error) {
	// Cancel both listeners with one source message
	var proxy chan struct{}
	if canceller != nil {
		proxy = make(chan struct{}, 2)
		go func() {
			<-canceller
			proxy <- struct{}{}
			proxy <- struct{}{}
		}()
	}

	err = Listen(pg_util.ListenOpts{
		Channel:   "thread.deleted",
		Canceller: proxy,
		OnMsg: func(msg string) (err error) {
			_, ints, err := SplitBoardAndInts(msg, 1)
			if err != nil {
				return
			}

			postCountCacheMu.Lock()
			delete(postCountCache, uint64(ints[0]))
			postCountCacheMu.Unlock()
			return
		},
	})
	if err != nil {
		return
	}

	return Listen(pg_util.ListenOpts{
		Channel:   "thread.new_post",
		Canceller: proxy,
		OnMsg: func(msg string) (err error) {
			retErr := func() error {
				return fmt.Errorf("invalid message: `%s`", msg)
			}

			split := strings.Split(msg, ",")
			if len(split) != 2 {
				return retErr()
			}
			id, err := strconv.ParseUint(split[0], 10, 64)
			if err != nil {
				return retErr()
			}
			postCount, err := strconv.ParseUint(split[1], 10, 64)
			if err != nil {
				return retErr()
			}

			postCountCacheMu.Lock()
			postCountCache[id] = postCount
			postCountCacheMu.Unlock()
			return
		},
	})
}

// Thread is a template for writing new threads to the database
type Thread struct {
	ID                   uint64
	PostCtr, ImageCtr    uint32
	UpdateTime, BumpTime int64
	Subject, Board       string
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
	if err == pgx.ErrNoRows {
		return false, nil
	}
	return
}

// InsertThread inserts a new thread into the database.
// Sets ID, OP and time on inserted post.
func InsertThread(tx *pgx.Tx, subject string, p *Post) (err error) {
	err = sq.Insert("threads").
		Columns("board", "subject").
		Values(p.Board, subject).
		Suffix("returning id").
		RunWith(tx).
		Scan(&p.ID)
	if err != nil {
		return
	}
	p.OP = p.ID
	return InsertPost(tx, p)
}

// WriteThread writes a thread and it's OP to the database. Only used for tests.
func WriteThread(tx *pgx.Tx, t Thread, p Post) (err error) {
	_, err = sq.
		Insert("threads").
		Columns("board", "id", "update_time", "bump_time", "subject").
		Values(
			t.Board,
			t.ID,
			t.UpdateTime,
			t.BumpTime,
			t.Subject,
		).
		RunWith(tx).
		Exec()
	if err != nil {
		return
	}
	return WritePost(tx, p)
}

// CheckThreadLocked checks, if a thread has been locked by a moderator
func CheckThreadLocked(id uint64) (locked bool, err error) {
	err = sq.Select("locked").
		From("threads").
		Where("id = ?", id).
		QueryRow().
		Scan(&locked)
	return
}
