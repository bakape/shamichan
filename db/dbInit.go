/*
 Initialises and loads RethinkDB
*/

package db

import (
	"fmt"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/util"
	r "github.com/dancannon/gorethink"
	"log"
)

const dbVersion = 2

// RSession exports the RethinkDB connection session. Used globally by the
// entire server.
var RSession *r.Session

// DB returns a function that creates a new DatabaseHelper. Used to simplify
// database queries.
// Simply DB()(${query}).${method}()
// Example: DB()(r.Table("posts").Get(1)).One(&Post)
func DB() func(r.Term) DatabaseHelper {
	return func(query r.Term) DatabaseHelper {
		return DatabaseHelper{query}
	}
}

// LoadDB establishes connections to RethinkDB and Redis and bootstraps both
// databases, if not yet done.
func LoadDB() {
	var err error
	RSession, err = r.Connect(r.ConnectOpts{
		Address: config.Config.Rethinkdb.Addr,
	})
	util.Throw(err)

	var isCreated bool
	DB()(r.DBList().Contains(config.Config.Rethinkdb.Db)).One(&isCreated)
	if isCreated {
		RSession.Use(config.Config.Rethinkdb.Db)
		verifyDBVersion()
	} else {
		initRethinkDB()
	}
}

// Confirm database verion is compatible, if not refuse to start, so we don't
// mess up the DB irreversably.
func verifyDBVersion() {
	var version int
	DB()(r.Table("main").Get("info").Field("dbVersion")).One(&version)
	if version != dbVersion {
		panic(fmt.Errorf("Incompatible RethinkDB database version: %d."+
			"See docs/migration.md", version))
	}
}

// Document is a eneric RethinkDB Document. For DRY-ness.
type Document struct {
	ID string `gorethink:"id"`
}

// All  tables needed for meguca operation
var AllTables = [...]string{"main", "threads", "posts", "images", "updates"}

// Central global information document
type infoDocument struct {
	Document
	DBVersion int `gorethink:"dbVersion"`

	// Is incremented on each new post. Ensures post number uniqueness
	PostCtr uint64 `gorethink:"postCtr"`
}

// Initialize a rethinkDB database
func initRethinkDB() {
	dbName := config.Config.Rethinkdb.Db
	log.Printf("Initialising database '%s'", dbName)
	DB()(r.DBCreate(dbName)).Exec()
	RSession.Use(dbName)
	CreateTables()
	DB()(r.Table("main").Insert([...]interface{}{
		infoDocument{Document{"info"}, dbVersion, 0},

		// History aka progress counters of boards, that get incremented on
		// post creation
		Document{"histCounts"},
	})).Exec()
	CreateIndeces()
}

// CreateTables creates all tables needed for meguca operation
func CreateTables() {
	for _, table := range AllTables {
		DB()(r.TableCreate(table)).Exec()
	}
}

// CreateIndeces create secondary indeces for faster table queries
func CreateIndeces() {
	DB()(r.Table("threads").IndexCreate("board")).Exec()
	for _, key := range [...]string{"op", "board"} {
		for _, table := range [...]string{"posts", "updates"} {
			DB()(r.Table(table).IndexCreate(key)).Exec()
		}
	}

	// Make sure all indeces are ready to avoid the race condition of and index
	// being accessed before its full creation.
	for _, table := range AllTables {
		DB()(r.Table(table).IndexWait()).Exec()
	}
}
