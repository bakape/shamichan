// Various periodic cleanup scripts and such

package db

import (
	"database/sql"
	"fmt"
	"math"
	"meguca/auth"
	"meguca/common"
	"meguca/config"
	"time"

	"github.com/go-playground/log"
)

// Run database clean up tasks at server start and regular intervals. Must be
// launched in separate goroutine.
func runCleanupTasks() {
	// To ensure even the once an hour tasks are run shortly after server start
	time.Sleep(time.Minute)
	runMinuteTasks()
	runHalfTasks()
	runHourTasks()

	min := time.Tick(time.Minute)
	half := time.Tick(time.Minute * 30)
	hour := time.Tick(time.Hour)
	for {
		select {
		case <-min:
			runMinuteTasks()
		case <-half:
			runHalfTasks()
		case <-hour:
			runHourTasks()
		}
	}
}

func runMinuteTasks() {
	logError("open post cleanup", closeDanglingPosts())
	expireRows("image_tokens", "bans")
}

func runHalfTasks() {
	logError("unrestrict pyu_limit", FreePyuLimit())
}

func runHourTasks() {
	expireRows("sessions")
	expireBy("created < now() at time zone 'utc' + '-7 days'",
		"mod_log", "reports")
	logError("remove identity info", removeIdentityInfo())
	logError("thread cleanup", deleteOldThreads())
	logError("board cleanup", deleteUnusedBoards())
	logError("image cleanup", deleteUnusedImages())
	logError("delete dangling open post bodies", cleanUpOpenPostBodies())
	_, err := db.Exec(`vacuum`)
	logError("vaccum database", err)
}

func logError(prefix string, err error) {
	if err != nil {
		log.Errorf("%s: %s: %#v", prefix, err, err)
	}
}

func expireBy(criterion string, tables ...string) {
	for _, t := range tables {
		_, err := sq.Delete(t).
			Where(criterion).
			Exec()
		if err != nil {
			logError(fmt.Sprintf("expiring table %s rows", t), err)
		}
	}
}

// Expire table rows by expiry timestamp
func expireRows(tables ...string) {
	expireBy("expires < now() at time zone 'utc'", tables...)
}

// Remove poster-identifying info from posts older than 7 days
func removeIdentityInfo() error {
	_, err := sq.Update("posts").
		Set("ip", nil).
		Where(`time < extract(epoch from now() at time zone 'utc'
			- interval '7 days')`).
		Exec()
	return err
}

// Close any open posts that have not been closed for 30 minutes
func closeDanglingPosts() error {
	r, err := sq.Select("id", "op", "board", "ip").
		From("posts").
		Where(`editing = true
			and time
				< floor(extract(epoch from now() at time zone 'utc')) - 900`).
		Query()
	if err != nil {
		return err
	}
	defer r.Close()

	type post struct {
		id, op uint64
		board  string
		ip     sql.NullString
	}

	posts := make([]post, 0, 8)
	var p post
	for r.Next() {
		err = r.Scan(&p.id, &p.op, &p.board, &p.ip)
		if err != nil {
			return err
		}
		posts = append(posts, p)
	}
	err = r.Err()
	if err != nil {
		return err
	}

	for _, p := range posts {
		// Get post body from BoltDB
		body, err := GetOpenBody(p.id)
		if err != nil {
			return err
		}

		links, com, err := common.ParseBody([]byte(body), p.board, p.id, p.ip.String, true)
		if err != nil {
			return err
		}
		err = ClosePost(p.id, p.op, body, links, com)
		if err != nil {
			return err
		}
	}

	return nil
}

// Delete boards that are older than N days and have not had any new posts for
// N days.
func deleteUnusedBoards() error {
	conf := config.Get()
	if !conf.PruneBoards {
		return nil
	}
	min := time.Now().Add(-time.Duration(conf.BoardExpiry) * time.Hour * 24)
	return InTransaction(func(tx *sql.Tx) (err error) {
		// Get all inactive boards
		var boards []string
		r, err := sq.Select("id").
			From("boards").
			Where(`created < ?
				and id != 'all'
				and (select coalesce(max(replyTime), 0)
						from threads
						where board = boards.id
					) < ?`,
				min, min.Unix(),
			).
			Query()
		if err != nil {
			return
		}
		var board string
		for r.Next() {
			err = r.Scan(&board)
			if err != nil {
				return
			}
			boards = append(boards, board)
		}
		err = r.Err()
		if err != nil {
			return
		}

		// Delete them and log to global moderation log
		for _, b := range boards {
			err = deleteBoard(tx, b, "system",
				fmt.Sprintf("board %s deleted for inactivity", b))
			if err != nil {
				return
			}
		}
		return
	})
}

func deleteBoard(tx *sql.Tx, id, by, reason string) (err error) {
	err = withTransaction(tx, sq.Delete("boards").Where("id = ?", id)).
		Exec()
	if err != nil {
		return
	}
	err = logModerationA(tx, auth.ModLogEntry{
		Type:   auth.DeleteBoard,
		Board:  "all",
		By:     by,
		Reason: reason,
	})
	if err != nil {
		return
	}
	_, err = tx.Exec(`select pg_notify('board_updated', $1)`, id)
	return
}

// Delete stale threads. Thread retention measured in a bumptime threshold, that
// is calculated as a function of post count till bump limit with an N days
// floor and ceiling.
func deleteOldThreads() (err error) {
	conf := config.Get()
	if !conf.PruneThreads {
		return
	}

	return InTransaction(func(tx *sql.Tx) (err error) {
		// Find threads to delete
		r, err := withTransaction(tx, sq.
			Select(
				"posts.id",
				"bumpTime",
				`(select count(*)
				from posts
				where posts.op = threads.id
			) as postCtr`,
				"posts.deleted",
			).
			From("threads").
			Join("posts on threads.id = posts.id"),
		).
			Query()
		if err != nil {
			return
		}
		defer r.Close()
		var (
			now         = time.Now().Unix()
			min         = float64(conf.ThreadExpiryMin * 24 * 3600)
			max         = float64(conf.ThreadExpiryMax * 24 * 3600)
			toDel       = make([]uint64, 0, 16)
			id, postCtr uint64
			bumpTime    int64
			deleted     sql.NullBool
		)
		for r.Next() {
			err = r.Scan(&id, &bumpTime, &postCtr, &deleted)
			if err != nil {
				return
			}
			threshold := min + (-max+min)*math.Pow(float64(postCtr)/3000-1, 3)
			if deleted.Bool {
				threshold /= 3
			}
			if threshold < min {
				threshold = min
			}
			if float64(now-bumpTime) > threshold {
				toDel = append(toDel, id)
			}
		}
		err = r.Err()
		if err != nil {
			return
		}

		var q *sql.Stmt
		if len(toDel) != 0 {
			// Deleted any matched threads
			q, err = tx.Prepare(`delete from threads
			where id = $1
			returning pg_notify('thread_deleted', board || ':' || id)`)
			if err != nil {
				return
			}
			for _, id := range toDel {
				_, err = q.Exec(id)
				if err != nil {
					return
				}
			}
		}

		return
	})
}
