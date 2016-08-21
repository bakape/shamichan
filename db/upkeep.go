// Various periodic cleanup scripts and such

package db

import (
	"log"
	"time"

	r "github.com/dancannon/gorethink"
)

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
					Field("editing"). // Older than 30 minutes
					And(post.Field("time").Lt(r.Now().ToEpochTime().Sub(1800)))
			}).
			Map(func(post r.Term) r.Term {
				return post.Field("id").CoerceTo("string")
			}).
			Do(func(ids r.Term) r.Term {
				return ids.
					Count().
					Eq(0).
					Branch(map[string]string{}, map[string]r.Term{
						"log": thread.
							Field("log").
							Append(ids.
								Map(func(id r.Term) r.Term {
									return r.Expr("06").
										Add(id).
										CoerceTo("binary")
								}).
								Reduce(func(a, b r.Term) r.Term {
									return a.Append(b)
								}),
							),
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

// Run database clean up tasks at server start and every 10 minutes
func runCleanupTasks() {
	timer10 := time.Tick(time.Minute * 10)
	timer1 := time.Tick(time.Minute * 1)
	for {
		select {
		case <-timer10:
			expireUserSessions()
		case <-timer1:
			closeDanglingPosts()
		}
	}
}

// Separate function, so we can test it
func expireUserSessions() {
	if err := Write(sessionExpiryQuery); err != nil {
		log.Printf("session cleanup: %s\n", err)
	}
}

// Close any open posts that have not been closed for 30 minutes
func closeDanglingPosts() {
	if err := Write(postClosingQuery); err != nil {
		log.Printf("open post cleanup: %s\n", err)
	}
}
