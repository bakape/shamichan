// Various periodic cleanup scripts and such

package db

import (
	"log"
	"time"

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
	Table("threads").
	Update(func(thread r.Term) r.Term {
		return thread.
			Field("posts").
			Values().
			Filter(func(post r.Term) r.Term {
				return post.
					Field("editing").
					Eq(true).
					And(post.
						Field("time"). // Older than 30 minutes
						Lt(r.Now().ToEpochTime().Sub(1800)),
					)
			}).
			Map(func(post r.Term) r.Term {
				return post.Field("id").CoerceTo("string")
			}).
			Do(func(ids r.Term) r.Term {
				return ids.
					Count().
					Eq(0).
					Branch(map[string]string{}, map[string]r.Term{
						"log": ids.
							Map(func(id r.Term) r.Term {
								return r.Expr("06").Add(id).CoerceTo("binary")
							}).
							Fold(thread.Field("log"), func(a, b r.Term) r.Term {
								return a.Append(b)
							}),
						"posts": ids.
							Map(func(id r.Term) interface{} {
								return []interface{}{
									id,
									map[string]bool{
										"editing": false,
									},
								}
							}).
							CoerceTo("object"),
					})
			})
	})

var getExpiredBoards = r.
	Table("boards").
	Filter(func(board r.Term) r.Term {
		return board.
			Field("created").
			Lt(r.Now().Sub(week)).
			And(r.
				Table("threads").
				GetAllByIndex("board", board.Field("id")).
				Field("posts").
				Map(func(posts r.Term) r.Term {
					return posts.
						CoerceTo("array").
						Nth(-1).
						AtIndex(1).
						Pluck("time")
				}).
				OrderBy("time").
				Nth(-1).
				Field("time").
				Lt(r.Now().ToEpochTime().Sub(week)),
			)
	}).
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
	var expired []string
	if err := All(getExpiredBoards, &expired); err != nil {
		return err
	}

	for _, board := range expired {
		q := r.Table("boards").Get(board).Delete()
		if err := Write(q); err != nil {
			return err
		}

		var threads []int64
		q = r.Table("threads").GetAllByIndex("board", board).Field("id")
		if err := All(q, &threads); err != nil {
			return err
		}
		for _, thread := range threads {
			if err := DeleteThread(thread); err != nil {
				return err
			}
		}
	}

	return nil
}

// DeleteThread dletes a thread from the database and dealocated any freed up
// images
func DeleteThread(id int64) error {
	q := r.
		Table("threads").
		Get(id).
		Delete(r.DeleteOpts{
			ReturnChanges: true,
		}).
		Field("changes").
		AtIndex(0).
		Field("old_val").
		Field("posts").
		Values().
		Filter(r.Row.HasFields("image")).
		Map(r.Row.Field("image").Field("SHA1")).
		Default([]string{}) // If aready deleted by another backend instance
	var images []string
	if err := All(q, &images); err != nil {
		return err
	}

	for _, sha1 := range images {
		if err := DeallocateImage(sha1); err != nil {
			return err
		}
	}

	return nil
}
