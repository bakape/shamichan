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

// Run database clean up tasks at server start and every 10 minutes
func runCleanupTasks() {
	timer := time.Tick(time.Minute * 10)
	for {
		expireUserSessions()
		<-timer
	}
}

// Separate function, so we can test it
func expireUserSessions() {
	if err := Write(sessionExpiryQuery); err != nil {
		log.Printf("session cleanup error: %s\n", err)
	}
}
