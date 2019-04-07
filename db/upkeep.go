// Various periodic cleanup scripts and such

package db

import (
	"database/sql"
	"fmt"
	"math"
	"time"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
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
	if config.ImagerMode != config.ImagerOnly {
		logError("open post cleanup", closeDanglingPosts())
		expireRows("image_tokens", "bans", "failed_captchas")
	}
}

func runHalfTasks() {
	if config.ImagerMode != config.ImagerOnly {
		logError("unrestrict pyu_limit", FreePyuLimit())
		logError("expire spam scores", expireSpamScores())
		logError("expire last solved captcha times", expireLastSolvedCaptchas())
	}
}

func runHourTasks() {
	if config.ImagerMode != config.ImagerOnly {
		expireRows("sessions")
		expireBy("created < now() at time zone 'utc' + '-7 days'",
			"mod_log", "reports")
		logError("remove identity info", removeIdentityInfo())
		logError("thread cleanup", deleteOldThreads())
		logError("board cleanup", deleteUnusedBoards())
		logError("delete dangling open post bodies", cleanUpOpenPostBodies())
		_, err := db.Exec(`vacuum`)
		logError("vaccum database", err)
	}
	if config.ImagerMode != config.NoImager {
		logError("image cleanup", deleteUnusedImages())
	}
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
		Set("password", nil).
		Where(`time < extract(epoch from now() at time zone 'utc'
			- interval '7 days')`).
		Where("ip is not null").
		Exec()
	return err
}

// Close any open posts that have not been closed for 30 minutes
func closeDanglingPosts() error {
	type post struct {
		id, op uint64
		board  string
		ip     sql.NullString
	}
	var (
		posts = make([]post, 0, 8)
		p     post
	)
	err := queryAll(
		sq.Select("id", "op", "board", "ip").
			From("posts").
			Where(`editing = true and time
				< floor(extract(epoch from now() at time zone 'utc')) - 900`),
		func(r *sql.Rows) (err error) {
			err = r.Scan(&p.id, &p.op, &p.board, &p.ip)
			if err != nil {
				return err
			}
			posts = append(posts, p)
			return
		},
	)
	if err != nil {
		return err
	}

	for _, p := range posts {
		// Get post body from BoltDB
		body, err := GetOpenBody(p.id)
		if err != nil {
			return err
		}

		links, com, err := common.ParseBody([]byte(body), p.board, p.op, p.id, p.ip.String, true)
		// Still close posts on invalid input
		switch err.(type) {
		case nil:
		case common.StatusError:
			if err.(common.StatusError).Code != 400 {
				return err
			}
			err = nil
		default:
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
	return InTransaction(false, func(tx *sql.Tx) (err error) {
		// Get all inactive boards
		var (
			boards []string
			board  string
		)
		err = queryAll(
			sq.Select("id").
				From("boards").
				Where(`created < ?
					and id != 'all'
					and (select coalesce(max(bumpTime), 0)
							from threads
							where board = boards.id
						) < ?`,
					min, min.Unix(),
				),
			func(r *sql.Rows) (err error) {
				err = r.Scan(&board)
				if err != nil {
					return
				}
				boards = append(boards, board)
				return
			},
		)
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
	_, err = sq.Delete("boards").Where("id = ?", id).RunWith(tx).Exec()
	if err != nil {
		return
	}
	err = logModeration(tx, auth.ModLogEntry{
		ModerationEntry: common.ModerationEntry{
			Type: common.DeleteBoard,
			By:   by,
			Data: reason,
		},
		Board: "all",
	})
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

	return InTransaction(false, func(tx *sql.Tx) (err error) {
		// Find threads to delete
		var (
			now         = time.Now().Unix()
			min         = float64(conf.ThreadExpiryMin * 24 * 3600)
			max         = float64(conf.ThreadExpiryMax * 24 * 3600)
			toDel       = make([]uint64, 0, 16)
			id, postCtr uint64
			bumpTime    int64
			deleted     sql.NullBool
		)
		err = queryAll(
			sq.
				Select(
					"threads.id",
					"bumpTime",
					`(select count(*)
						from posts
						where posts.op = threads.id
						) as postCtr`,
					fmt.Sprintf(
						`(select exists (
							select 1 from post_moderation
							where post_id = threads.id and type = %d))`,
						common.DeletePost),
				).
				From("threads").
				Join("posts on threads.id = posts.id").
				RunWith(tx),
			func(r *sql.Rows) (err error) {
				err = r.Scan(&id, &bumpTime, &postCtr, &deleted)
				if err != nil {
					return
				}
				threshold := min +
					(-max+min)*math.Pow(float64(postCtr)/common.BumpLimit-1, 3)
				if deleted.Bool {
					threshold /= 3
				}
				if threshold < min {
					threshold = min
				}
				if float64(now-bumpTime) > threshold {
					toDel = append(toDel, id)
				}
				return
			},
		)
		if err != nil {
			return
		}

		var q *sql.Stmt
		if len(toDel) != 0 {
			// Deleted any matched threads
			q, err = tx.Prepare(`delete from threads where id = $1`)
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
