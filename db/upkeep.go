// Various periodic cleanup scripts and such

package db

import (
	"time"

	"github.com/go-playground/log"
)

// Run database clean up tasks at server start and regular intervals. Must be
// launched in separate goroutine.
func runCleanupTasks() {
	sec := time.Tick(time.Second)
	min := time.Tick(time.Minute)
	hour := time.Tick(time.Hour)

	// To ensure even the once an hour tasks are run shortly after server start
	go func() {
		time.Sleep(time.Minute)
		runHourTasks()
	}()

	for {
		select {
		case <-sec:
			// TODO
			// logError("flush open post bodies", FlushOpenPostBodies())
			logError("spam score buffer flush", syncSpamScores)
		case <-min:
			// TODO:
			// logError("open post cleanup", closeDanglingPosts())

			logError("expired row cleanup", func() error {
				_, err := db.Exec(`delete from expiries where expires < now()`)
				return err
			})
		case <-hour:
			runHourTasks()
		}
	}
}

func runHourTasks() {
	// TODO
	// logError("thread cleanup", deleteOldThreads())
	// logError("image cleanup", deleteUnusedImages())
}

func logError(prefix string, fn func() error) {
	err := fn()
	if err != nil {
		log.Errorf("%s: %s: %#v", prefix, err, err)
	}
}

// // Close any open posts that have not been closed for 30 minutes
// func closeDanglingPosts() (err error) {
// 	type post struct {
// 		id          uint64
// 		board, body string
// 	}

// 	r, err := db.Query(
// 		`select id, body
// 		from posts
// 		where editing = true
// 			and time
// 				< floor(extract(epoch from now() at time zone 'utc')) - 900
// 		order by id`,
// 	)
// 	if err != nil {
// 		return
// 	}
// 	defer r.Close()

// 	var (
// 		posts []post
// 		p     post
// 	)
// 	for r.Next() {
// 		err = r.Scan(&p.id, &p.board, &p.body)
// 		if err != nil {
// 			return err
// 		}
// 		posts = append(posts, p)
// 	}
// 	err = r.Err()
// 	if err != nil {
// 		return
// 	}

// 	for _, p := range posts {
// 		links, com, err := parser.ParseBody([]byte(p.body), true)
// 		switch err.(type) {
// 		case nil:
// 		case common.StatusError:
// 			// Still close posts on invalid input
// 			if err.(common.StatusError).Code != 400 {
// 				return err
// 			}
// 			err = nil
// 		default:
// 			return err
// 		}
// 		err = ClosePost(p.id, p.board, p.body, links, com)
// 		if err != nil {
// 			return err
// 		}
// 	}

// 	return nil
// }

// // Delete stale threads. Thread retention measured in a bump time threshold,
// // that is calculated as a function of post count till bump limit with an N days
// // floor and ceiling.
// func deleteOldThreads() (err error) {
// 	conf := config.Get()
// 	if !conf.PruneThreads {
// 		return
// 	}

// 	return InTransaction(func(tx *pgx.Tx) (err error) {
// 		// TODO: Store all dates as timestamptz and simplify this deletion code

// 		// Find threads to delete
// 		r, err := db.
// 			Query(
// 				fmt.Sprintf(
// 					`delete from threads
// 					where bump_time < (now() - )
// 						t.id,
// 						bump_time,
// 						post_count(t.id),
// 						(
// 							select exists (
// 								select
// 								from post_moderation
// 								where post_id = t.id and type = %d
// 							)
// 						)
// 					from threads t
// 					join posts p on t.id = p.id`,
// 					common.DeletePost,
// 				),
// 			)
// 		if err != nil {
// 			return
// 		}

// 		var (
// 			now           = time.Now().Unix()
// 			expiry        = float64(*24 * 3600)
// 			toDel         = make([]uint64, 0, 16)
// 			id, postCount uint64
// 			bumpTime      int64
// 			deleted       sql.NullBool
// 		)
// 		for r.Next() {
// 			err = r.Scan(&id, &bumpTime, &postCount, &deleted)
// 			if err != nil {
// 				return
// 			}
// 			threshold := min + (-max+min)*math.Pow(float64(postCount)/common.BumpLimit-1, 3)
// 			if deleted.Bool {
// 				threshold /= 3
// 			}
// 			if threshold < min {
// 				threshold = min
// 			}
// 			if float64(now-bumpTime) > threshold {
// 				toDel = append(toDel, id)
// 			}
// 		}

// 		var q *sql.Stmt
// 		if len(toDel) != 0 {
// 			// Deleted any matched threads
// 			q, err = tx.Prepare(`delete from threads where id = $1`)
// 			if err != nil {
// 				return
// 			}
// 			for _, id := range toDel {
// 				_, err = q.Exec(id)
// 				if err != nil {
// 					return
// 				}
// 			}
// 		}

// 		return
// 	})
// }
