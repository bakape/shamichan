/*
 Initialises and loads Redis and RethinkDB
*/

package server

import (
	"fmt"
	r "github.com/dancannon/gorethink"
)

const dbVersion = 2

// rSession exports the RethinkDB connection session
var rSession *r.Session

// loadDB establishes connections to RethinkDB and Redis and bootstraps both
// databases, if not yet done.
func loadDB() {
	var err error
	rSession, err = r.Connect(r.ConnectOpts{
		Address: config.Rethinkdb.Addr,
	})
	throw(err)

	// Assign the database helper function. Tests will implement and assign
	// their own
	db = func() func(r.Term) Database {
		return func(query r.Term) Database {
			return DatabaseHelper{query}
		}
	}

	var isCreated bool
	db()(r.DBList().Contains(config.Rethinkdb.Db)).One(&isCreated)
	if !isCreated {
		initRethinkDB()
	} else {
		verifyDBVersion()
	}
}

// Confirm database verion is compatible, if not refuse to start, so we don't
// mess up the DB irreversably.
func verifyDBVersion() {
	rSession.Use(config.Rethinkdb.Db)
	var version int
	db()(r.Table("main").Get("info").Field("dbVersion")).One(&version)
	if version != dbVersion {
		panic(fmt.Sprintf("Incompatible RethinkDB database version: %d."+
			"See docs/migration.md", version))
	}
}

// Document is a eneric RethinkDB Document. For DRY-ness.
type Document struct {
	ID string `gorethink:"id"`
}

func initRethinkDB() {
	db()(r.DBCreate(config.Rethinkdb.Db)).Exec()
	rSession.Use(config.Rethinkdb.Db)
	tables := [...]string{"main", "threads", "posts", "images", "updates"}
	for _, table := range tables {
		db()(r.TableCreate(table)).Exec()
	}
	db()(r.Table("main").Insert([...]interface{}{
		struct {
			Document
			DBVersion int `gorethink:"dbVersion"`

			// Is incremented on each new post. Ensures post number uniqueness
			PostCtr uint64 `gorethink:"postCtr"`
		}{
			Document{"info"}, dbVersion, 0,
		},

		// History aka progress counters of boards, that get incremented on
		// post creation
		Document{"histCounts"},
	})).Exec()

	// Create secondary indeces
	db()(r.Table("threads").IndexCreate("board")).Exec()
	for _, key := range [...]string{"op", "board"} {
		for _, table := range [...]string{"posts", "updates"} {
			db()(r.Table(table).IndexCreate(key)).Exec()
		}
	}
}
