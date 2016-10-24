// Various periodic cleanup scripts and such

package db

import (
	"log"
	"time"

	"github.com/bakape/meguca/config"
	r "github.com/dancannon/gorethink"
)

const week = 7 * 24 * 60 * 60

var sessionExpiryQuery = r.
	Table("accounts").
	Update(map[string]r.Term{
		"sessions": r.Row.
			Field("sessions").
			Filter(func(s r.Term) r.Term {
				return s.Field("expires").Gt(r.Now())
			}),
	})

var postClosingQuery = r.
	Table("posts").
	GetAllByIndex("editing", true). // Older than 30 minutes
	Filter(r.Row.Field("time").Lt(r.Now().ToEpochTime().Sub(1800))).
	Update(map[string]interface{}{
		"log": r.Row.Field("log").Append(r.
			Expr("06").
			Add(r.Row.Field("id").CoerceTo("string")).
			CoerceTo("binary"),
		),
		"editing":     false,
		"lastUpdated": r.Now().ToEpochTime().Floor(),
	})

var getExpiredBoards = r.
	Table("boards").
	Filter(r.
		Row.
		Field("created").
		Lt(r.Now().Sub(week)).
		And(r.
			Table("posts").
			GetAllByIndex("board", r.Row.Field("id")).
			Pluck("time").
			OrderBy("time").
			Nth(-1).
			Field("time").
			Lt(r.Now().ToEpochTime().Sub(week)).
			Default(true),
		),
	).
	Field("id")

var expireImageTokensQuery = r.
	Table("imageTokens").
	Filter(r.Row.Field("expires").Lt(r.Now())).
	Delete(r.DeleteOpts{ReturnChanges: true}).
	Do(func(d r.Term) r.Term {
		return d.
			Field("deleted").
			Eq(0).
			Branch(
				r.Expr([]string{}),
				d.
					Field("changes").
					Field("old_val").
					Field("SHA1"),
			)
	})

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
}

func runHourTasks() {
	logError("session cleanup", expireUserSessions())
	logError("board cleanup", deleteUnusedBoards())
}

func logError(prefix string, err error) {
	if err != nil {
		log.Printf("%s: %s\n", prefix, err)
	}
}

// Separate function, so we can test it
func expireUserSessions() error {
	return Write(sessionExpiryQuery)
}

// Close any open posts that have not been closed for 30 minutes
func closeDanglingPosts() error {
	return Write(postClosingQuery)
}

// Remove any expired image tokens and decrement or dealocate their target
// image's assets
func expireImageTokens() error {
	var toDealloc []string
	if err := All(expireImageTokensQuery, &toDealloc); err != nil {
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
// a week.
func deleteUnusedBoards() error {
	if !config.Get().PruneBoards {
		return nil
	}

	var expired []string
	if err := All(getExpiredBoards, &expired); err != nil {
		return err
	}

	for _, board := range expired {
		var threads []int64
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
		// less consequences to an interupted cleanup task.
		q = r.Table("boards").Get(board).Delete()
		if err := Write(q); err != nil {
			return err
		}
	}

	return nil
}

// DeleteThread deletes a thread from the database and dealocated any freed up
// images
func DeleteThread(id int64) error {
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
		Default("") // Aready deleted by another backend instance or no image
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
