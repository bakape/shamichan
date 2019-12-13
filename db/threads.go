package db

import (
	"github.com/bakape/meguca/auth"
	"github.com/jackc/pgx"
)

// var (
// 	postCountCache           = make(map[uint64]uint64)
// 	postCountCacheMu         sync.RWMutex
// 	errTooManyWatchedThreads = common.StatusError{
// 		Err:  errors.New("too many watched threads"),
// 		Code: 400,
// 	}
// )

// // Diff of passed and actual thread posts counts
// type ThreadPostCountDiff struct {
// 	Changed map[uint64]uint64 `json:"changed"`
// 	Deleted []uint64          `json:"deleted"`
// }

// // Return diff of passed and actual thread post counts
// func DiffThreadPostCounts(old map[uint64]uint64) (
// 	diff ThreadPostCountDiff, err error,
// ) {
// 	if len(old) > 1000 {
// 		err = errTooManyWatchedThreads
// 		return
// 	}

// 	postCountCacheMu.RLock()
// 	defer postCountCacheMu.RUnlock()

// 	diff.Changed = make(map[uint64]uint64, len(old))
// 	diff.Deleted = make([]uint64, 0)
// 	for thread, count := range old {
// 		actual, ok := postCountCache[thread]
// 		if !ok {
// 			diff.Deleted = append(diff.Deleted, thread)
// 		} else if actual != count {
// 			diff.Changed[thread] = actual
// 		}
// 	}

// 	return
// }

// func loadThreadPostCounts() (err error) {
// 	err = readThreadPostCounts()
// 	if err != nil {
// 		return
// 	}
// 	return listenForThreadUpdates(nil)
// }

// func readThreadPostCounts() (err error) {
// 	r, err := db.Query(
// 		`select op, count(*)
// 		from posts
// 		group by op`,
// 	)
// 	if err != nil {
// 		return
// 	}
// 	defer r.Close()

// 	postCountCacheMu.Lock()
// 	defer postCountCacheMu.Unlock()

// 	var thread, postCount uint64
// 	for r.Next() {
// 		err = r.Scan(&thread, &postCount)
// 		if err != nil {
// 			return
// 		}
// 		postCountCache[thread] = postCount
// 	}
// 	return r.Err()
// }

// // Separate function for easier testing
// func listenForThreadUpdates(ctx context.Context) (err error) {
// 	err = Listen(pg_util.ListenOpts{
// 		Channel: "thread.deleted",
// 		Context: ctx,
// 		OnMsg: func(msg string) (err error) {
// 			thread, err := strconv.ParseUint(msg, 10, 64)
// 			if err != nil {
// 				return
// 			}

// 			postCountCacheMu.Lock()
// 			delete(postCountCache, thread)
// 			postCountCacheMu.Unlock()
// 			return
// 		},
// 	})
// 	if err != nil {
// 		return
// 	}

// 	return Listen(pg_util.ListenOpts{
// 		Channel: "thread.new_post",
// 		Context: ctx,
// 		OnMsg: func(msg string) (err error) {
// 			retErr := func() error {
// 				return fmt.Errorf("invalid message: `%s`", msg)
// 			}

// 			split := strings.Split(msg, ",")
// 			if len(split) != 2 {
// 				return retErr()
// 			}
// 			id, err := strconv.ParseUint(split[0], 10, 64)
// 			if err != nil {
// 				return retErr()
// 			}
// 			postCount, err := strconv.ParseUint(split[1], 10, 64)
// 			if err != nil {
// 				return retErr()
// 			}

// 			postCountCacheMu.Lock()
// 			postCountCache[id] = postCount
// 			postCountCacheMu.Unlock()
// 			return
// 		},
// 	})
// }

// Insert thread and empty post into DB and return the post ID
func InsertThread(subject string, tags []string, authKey auth.Token) (
	id uint64, err error,
) {
	err = InTransaction(func(tx *pgx.Tx) (err error) {
		err = tx.
			QueryRow(
				`insert into threads (subject, tags)
				values ($1, $2)
				returning id`,
				subject,
				tags,
			).
			Scan(&id)
		if err != nil {
			return
		}

		_, err = tx.Exec(
			`insert into posts (id, thread, auth_key)
			values ($1, $1, $2)`,
			id,
			authKey,
		)
		return
	})
	return
}

// Check, if thread exists in the database
func ThreadExists(id uint64) (exists bool, err error) {
	err = db.
		QueryRow(
			`select exists (
				select
				from threads
				where id = $1
			)`,
			id,
		).
		Scan(&exists)
	return
}
