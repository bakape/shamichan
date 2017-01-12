// Various periodic cleanup scripts and such

package db

import (
	"log"
	"time"

	"github.com/bakape/meguca/common"
)

// // Remove any identity information from post after a week. Also clear the log,
// // as it will most likely be pointless by then.
// var postCleanupQ = r.
// 	Table("posts").
// 	Filter(r.Row.HasFields("ip")).
// 	Filter(timeFilter(day * 7)).
// 	Replace(r.Row.Without("ip", "password").Merge(map[string][]string{
// 		"log": []string{},
// 	}))

// var expireBansQ = r.
// 	Table("bans").
// 	Between(r.MinVal, r.Now(), r.BetweenOpts{
// 		Index: "expires",
// 	}).
// 	Delete()

// Run database clean up tasks at server start and regular intervals. Must be
// launched in separate goroutine.
func runCleanupTasks() {
	// To ensure even the once an hour tasks are run shortly after server start
	time.Sleep(time.Minute)
	runMinuteTasks()
	runHourTasks()

	min := time.Tick(time.Minute)
	hour := time.Tick(time.Hour)
	for {
		select {
		case <-min:
			runMinuteTasks()
		case <-hour:
			runHourTasks()
		}
	}
}

func runMinuteTasks() {
	logError("open post cleanup", closeDanglingPosts)
	logError("expire image tokens", expireImageTokens)
	// logError("expire bans", Write(expireBansQ))
}

func runHourTasks() {
	logError("session cleanup", expireUserSessions)
	// logError("board cleanup", deleteUnusedBoards())
	// logError("thread cleanup", deleteOldThreads())
	// logError("old post cleanup", Write(postCleanupQ))
}

func logError(prefix string, fn func() error) {
	if err := fn(); err != nil {
		log.Printf("%s: %s\n", prefix, err)
	}
}

// Close any open posts that have not been closed for 30 minutes
func closeDanglingPosts() (err error) {
	tx, err := db.Begin()
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Read and close all expired posts
	r, err := tx.Stmt(prepared["closeExpiredOpenPosts"]).Query()
	if err != nil {
		return
	}

	type post struct {
		id, op uint64
	}

	posts := make([]post, 0, 8)
	for r.Next() {
		var p post
		err = r.Scan(&p.id, &p.op)
		if err != nil {
			return
		}
		posts = append(posts, p)
	}

	// Write updates to the replication log
	q := tx.Stmt(prepared["updateLog"])
	for _, p := range posts {
		var msg []byte
		msg, err = common.EncodeMessage(common.MessageClosePost, p.id)
		if err != nil {
			return
		}
		_, err = q.Exec(p.op, msg)
		if err != nil {
			return
		}
	}

	return tx.Commit()
}

// // Delete boards that are older than 1 week and have not had any new posts for
// // N days.
// func deleteUnusedBoards() error {
// 	conf := config.Get()
// 	if !conf.PruneBoards {
// 		return nil
// 	}

// 	q := r.
// 		Table("boards").
// 		Filter(r.
// 			Row.
// 			Field("created").
// 			Lt(r.Now().Sub(day * conf.BoardExpiry)).
// 			And(r.
// 				Table("posts").
// 				GetAllByIndex("board", r.Row.Field("id")).
// 				Pluck("time").
// 				OrderBy("time").
// 				Nth(-1).
// 				Field("time").
// 				Lt(r.Now().ToEpochTime().Sub(day * conf.BoardExpiry)).
// 				Default(true),
// 			),
// 		).
// 		Field("id")

// 	var expired []string
// 	if err := All(q, &expired); err != nil {
// 		return err
// 	}

// 	for _, board := range expired {
// 		if err := DeleteBoard(board); err != nil {
// 			return err
// 		}
// 	}

// 	return nil
// }

// // DeleteBoard deletes a board and all of its contained threads and posts
// func DeleteBoard(board string) error {
// 	var threads []uint64
// 	q := r.Table("threads").GetAllByIndex("board", board).Field("id")
// 	if err := All(q, &threads); err != nil {
// 		return err
// 	}

// 	for _, thread := range threads {
// 		if err := DeleteThread(thread); err != nil {
// 			return err
// 		}
// 	}

// 	// Perform board deletion after all threads are deleted, so there are
// 	// less consequences to an interrupted cleanup task.
// 	q = r.Table("boards").Get(board).Delete()
// 	if err := Write(q); err != nil {
// 		return err
// 	}

// 	return nil
// }

// // Delete threads that have not had any new posts in N days.
// func deleteOldThreads() error {
// 	conf := config.Get()
// 	if !conf.PruneThreads {
// 		return nil
// 	}

// 	q := r.
// 		Table("posts").
// 		GroupByIndex("op").
// 		Field("time").
// 		Max().
// 		Ungroup().
// 		Filter(r.Row.
// 			Field("reduction").
// 			Lt(r.Now().ToEpochTime().Sub(day * conf.ThreadExpiry)),
// 		).
// 		Field("group").
// 		Default(nil)

// 	var expired []uint64
// 	if err := All(q, &expired); err != nil {
// 		return err
// 	}

// 	for _, t := range expired {
// 		if err := DeleteThread(t); err != nil {
// 			return err
// 		}
// 	}

// 	return nil
// }

// // DeleteThread deletes a thread from the database and deallocated any freed up
// // images
// func DeleteThread(id uint64) error {
// 	if err := Write(FindThread(id).Delete()); err != nil {
// 		return err
// 	}

// 	q := r.
// 		Table("posts").
// 		GetAllByIndex("op", id).
// 		Delete(r.DeleteOpts{
// 			ReturnChanges: true,
// 		}).
// 		Field("changes").
// 		Field("old_val").
// 		Field("image").
// 		Field("SHA1").
// 		Default("") // Already deleted by another backend instance or no image
// 	var images []string
// 	if err := All(q, &images); err != nil {
// 		return err
// 	}

// 	for _, sha1 := range images {
// 		if sha1 != "" {
// 			if err := DeallocateImage(sha1); err != nil {
// 				return err
// 			}
// 		}
// 	}

// 	return nil
// }
