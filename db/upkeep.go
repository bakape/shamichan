// Various periodic cleanup scripts and such

package db

import (
	"log"
	"time"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
)

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
	logError("open post cleanup", closeDanglingPosts())
	logPrepared("expire_image_tokens", "expire_bans")
}

func runHourTasks() {
	logPrepared("expire_user_sessions", "remove_identity_info")
	// logError("thread cleanup", deleteOldThreads())
	logError("board cleanup", deleteUnusedBoards())
}

func logPrepared(ids ...string) {
	for _, id := range ids {
		logError(id, execPrepared(id))
	}
}

func logError(prefix string, err error) {
	if err != nil {
		log.Printf("%s: %s\n", prefix, err)
	}
}

// Close any open posts that have not been closed for 30 minutes
func closeDanglingPosts() (err error) {
	tx, err := db.Begin()
	if err != nil {
		return
	}
	defer RollbackOnError(tx, &err)

	// Read and close all expired posts
	r, err := tx.Stmt(prepared["close_expired_open_posts"]).Query()
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
	q := tx.Stmt(prepared["update_log"])
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

// Delete boards that are older than N days and have not had any new posts for
// N days.
func deleteUnusedBoards() error {
	conf := config.Get()
	if !conf.PruneBoards {
		return nil
	}
	min := time.Now().Add(-time.Duration(conf.BoardExpiry) * time.Hour * 24)
	return execPrepared("delete_unused_boards", min)
}

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

// DeleteBoard deletes a board and all of its contained threads and posts
func DeleteBoard(board string) error {
	_, err := prepared["delete_board"].Exec(board)
	return err
}
