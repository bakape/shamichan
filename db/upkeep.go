// Various periodic cleanup scripts and such

package db

import (
	"log"
	"time"

	"github.com/bakape/meguca/config"
	r "github.com/dancannon/gorethink"
)

const day = 24 * 60 * 60

var sessionExpiryQ = r.
	Table("accounts").
	Update(map[string]r.Term{
		"sessions": r.Row.
			Field("sessions").
			Filter(func(s r.Term) r.Term {
				return s.Field("expires").Gt(r.Now())
			}),
	})

var postClosingQ = r.
	Table("posts").
	GetAllByIndex("editing", true). // Older than 30 minutes
	Filter(timeFilter(1800)).
	Update(map[string]interface{}{
		"log": r.Row.Field("log").Append(r.
			Expr("06").
			Add(r.Row.Field("id").CoerceTo("string")),
		),
		"editing":     false,
		"lastUpdated": r.Now().ToEpochTime().Floor(),
	})

// Remove any identity information from post after a week. Also clear the log,
// as it will most likely be pointless by then.
var postCleanupQ = r.
	Table("posts").
	Filter(r.Row.HasFields("ip")).
	Filter(timeFilter(day * 7)).
	Replace(r.Row.Without("ip", "password").Merge(map[string][]string{
		"log": []string{},
	}))

var expireImageTokensQ = r.
	Table("imageTokens").
	Between(r.MinVal, r.Now(), r.BetweenOpts{
		Index: "expires",
	}).
	Delete(r.DeleteOpts{ReturnChanges: true}).
	Do(func(d r.Term) r.Term {
		return d.Field("deleted").Eq(0).Branch(
			r.Expr([]string{}),
			d.Field("changes").Field("old_val").Field("SHA1"),
		)
	})

var expireBansQ = r.
	Table("bans").
	Between(r.MinVal, r.Now(), r.BetweenOpts{
		Index: "expires",
	}).
	Delete()

func timeFilter(sec int) r.Term {
	return r.Row.
		Field("time").
		Lt(r.Now().ToEpochTime().Floor().Sub(sec))
}

// Run database clean up tasks at server start and regular intervals. Must be
// launched in separate goroutine.
func runCleanupTasks() {
	// To ensure even the once an hour tasks are run shortly after server start
	time.Sleep(time.Minute)
	runMinuteTasks()
	runHourTasks()

	timerMin := time.Tick(time.Minute)
	timerHour := time.Tick(time.Hour)
	for {
		select {
		case <-timerMin:
			runMinuteTasks()
		case <-timerHour:
			runHourTasks()
		}
	}
}

func runMinuteTasks() {
	logError("open post cleanup", closeDanglingPosts())
	logError("expire image tokens", expireImageTokens())
	logError("expire bans", Write(expireBansQ))
}

func runHourTasks() {
	logError("session cleanup", expireUserSessions())
	logError("board cleanup", deleteUnusedBoards())
	logError("thread cleanup", deleteOldThreads())
	logError("old post cleanup", Write(postCleanupQ))
}

func logError(prefix string, err error) {
	if err != nil {
		log.Printf("%s: %s\n", prefix, err)
	}
}

// Separate function, so we can test it
func expireUserSessions() error {
	return Write(sessionExpiryQ)
}

// Close any open posts that have not been closed for 30 minutes
func closeDanglingPosts() error {
	return Write(postClosingQ)
}

// Remove any expired image tokens and decrement or deallocate their target
// image's assets
func expireImageTokens() error {
	var toDealloc []string
	if err := All(expireImageTokensQ, &toDealloc); err != nil {
		return err
	}

	for _, sha1 := range toDealloc {
		if err := DeallocateImage(sha1); err != nil {
			return err
		}
	}

	return nil
}

// Delete boards that are older than 1 week and have not had any new posts for
// N days.
func deleteUnusedBoards() error {
	conf := config.Get()
	if !conf.PruneBoards {
		return nil
	}

	q := r.
		Table("boards").
		Filter(r.
			Row.
			Field("created").
			Lt(r.Now().Sub(day * conf.BoardExpiry)).
			And(r.
				Table("posts").
				GetAllByIndex("board", r.Row.Field("id")).
				Pluck("time").
				OrderBy("time").
				Nth(-1).
				Field("time").
				Lt(r.Now().ToEpochTime().Sub(day * conf.BoardExpiry)).
				Default(true),
			),
		).
		Field("id")

	var expired []string
	if err := All(q, &expired); err != nil {
		return err
	}

	for _, board := range expired {
		if err := DeleteBoard(board); err != nil {
			return err
		}
	}

	return nil
}

// DeleteBoard deletes a board and all of its contained threads and posts
func DeleteBoard(board string) error {
	var threads []uint64
	q := r.Table("threads").GetAllByIndex("board", board).Field("id")
	if err := All(q, &threads); err != nil {
		return err
	}

	for _, thread := range threads {
		if err := DeleteThread(thread); err != nil {
			return err
		}
	}

	// Perform board deletion after all threads are deleted, so there are
	// less consequences to an interrupted cleanup task.
	q = r.Table("boards").Get(board).Delete()
	if err := Write(q); err != nil {
		return err
	}

	return nil
}

// Delete threads that have not had any new posts in N days.
func deleteOldThreads() error {
	conf := config.Get()
	if !conf.PruneThreads {
		return nil
	}

	q := r.
		Table("posts").
		GroupByIndex("op").
		Field("time").
		Max().
		Ungroup().
		Filter(r.Row.
			Field("reduction").
			Lt(r.Now().ToEpochTime().Sub(day * conf.ThreadExpiry)),
		).
		Field("group").
		Default(nil)

	var expired []uint64
	if err := All(q, &expired); err != nil {
		return err
	}

	for _, t := range expired {
		if err := DeleteThread(t); err != nil {
			return err
		}
	}

	return nil
}

// DeleteThread deletes a thread from the database and deallocated any freed up
// images
func DeleteThread(id uint64) error {
	if err := Write(FindThread(id).Delete()); err != nil {
		return err
	}

	q := r.
		Table("posts").
		GetAllByIndex("op", id).
		Delete(r.DeleteOpts{
			ReturnChanges: true,
		}).
		Field("changes").
		Field("old_val").
		Field("image").
		Field("SHA1").
		Default("") // Already deleted by another backend instance or no image
	var images []string
	if err := All(q, &images); err != nil {
		return err
	}

	for _, sha1 := range images {
		if sha1 != "" {
			if err := DeallocateImage(sha1); err != nil {
				return err
			}
		}
	}

	return nil
}
