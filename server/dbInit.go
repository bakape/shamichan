/*
 Initialises and loads Redis and RethinkDB
*/

package server

import (
	"fmt"
	r "github.com/dancannon/gorethink"
	"log"
)

const dbVersion = 2

// rSession exports the RethinkDB connection session
var rSession *r.Session

// db returns a function that creates a new DatabaseHelper. Used to simplify
// database queries.
// Simply db()(${query}).${method}()
// Example: db()(r.Table("posts").Get(1)).One(&Post)
func db() func(r.Term) DatabaseHelper {
	return func(query r.Term) DatabaseHelper {
		return DatabaseHelper{query}
	}
}

// loadDB establishes connections to RethinkDB and Redis and bootstraps both
// databases, if not yet done.
func loadDB() {
	var err error
	rSession, err = r.Connect(r.ConnectOpts{
		Address: config.Rethinkdb.Addr,
	})
	throw(err)

	var isCreated bool
	db()(r.DBList().Contains(config.Rethinkdb.Db)).One(&isCreated)
	if isCreated {
		rSession.Use(config.Rethinkdb.Db)
		verifyDBVersion()
	} else {
		initRethinkDB()
	}
}

// Confirm database verion is compatible, if not refuse to start, so we don't
// mess up the DB irreversably.
func verifyDBVersion() {
	var version int
	db()(r.Table("main").Get("info").Field("dbVersion")).One(&version)
	if version != dbVersion {
		panic(fmt.Errorf("Incompatible RethinkDB database version: %d."+
			"See docs/migration.md", version))
	}
}

// Document is a eneric RethinkDB Document. For DRY-ness.
type Document struct {
	ID string `gorethink:"id"`
}

var allTables = [...]string{"main", "threads", "posts", "images", "updates"}

// Central global information document
type infoDocument struct {
	Document
	DBVersion int `gorethink:"dbVersion"`

	// Is incremented on each new post. Ensures post number uniqueness
	PostCtr uint64 `gorethink:"postCtr"`
}

// Initialize a rethinkDB database
func initRethinkDB() {
	dbName := config.Rethinkdb.Db
	log.Printf("Initialising database '%s'", dbName)
	db()(r.DBCreate(dbName)).Exec()
	rSession.Use(dbName)
	createTables()
	db()(r.Table("main").Insert([...]interface{}{
		infoDocument{Document{"info"}, dbVersion, 0},

		// History aka progress counters of boards, that get incremented on
		// post creation
		Document{"histCounts"},
	})).Exec()
	createIndeces()
}

// Create all tables needed for meguca operation
func createTables() {
	for _, table := range allTables {
		db()(r.TableCreate(table)).Exec()
	}
}

// Create secondary indeces for faster table queries
func createIndeces() {
	db()(r.Table("threads").IndexCreate("board")).Exec()
	for _, key := range [...]string{"op", "board"} {
		for _, table := range [...]string{"posts", "updates"} {
			db()(r.Table(table).IndexCreate(key)).Exec()
		}
	}

	// Make sure all indeces are ready to avoid the race condition of and index
	// being accessed before its full creation.
	for _, table := range allTables {
		db()(r.Table(table).IndexWait()).Exec()
	}
}
