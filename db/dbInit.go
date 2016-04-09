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

// DB creates a new DatabaseHelper. Used to simplify database queries.
// Example: err := DB(r.Table("posts").Get(1)).One(&Post)
func DB(query r.Term) DatabaseHelper {
	return DatabaseHelper{query}
}

// LoadDB establishes connections to RethinkDB and Redis and bootstraps both
// databases, if not yet done.
func LoadDB() (err error) {
	RSession, err = r.Connect(r.ConnectOpts{
		Address: config.Config.Rethinkdb.Addr,
	})
	if err != nil {
		return util.WrapError("Error connecting to RethinkDB", err)
	}

	var isCreated bool
	err = DB(r.DBList().Contains(config.Config.Rethinkdb.Db)).One(&isCreated)
	if err != nil {
		return util.WrapError("Error checking, if database exists", err)
	}
	if isCreated {
		RSession.Use(config.Config.Rethinkdb.Db)
		return verifyDBVersion()
	}
	return initRethinkDB()
}

// Confirm database verion is compatible, if not refuse to start, so we don't
// mess up the DB irreversably.
func verifyDBVersion() error {
	var version int
	err := DB(r.Table("main").Get("info").Field("dbVersion")).One(&version)
	if err != nil {
		return util.WrapError("Error reading database version", err)
	}
	if version != dbVersion {
		return fmt.Errorf(
			"Incompatible RethinkDB database version: %d. "+
				"See docs/migration.md",
			version,
		)
	}
	return nil
}

// Document is a eneric RethinkDB Document. For DRY-ness.
type Document struct {
	ID string `gorethink:"id"`
}

// AllTables are all tables needed for meguca operation
var AllTables = [...]string{"main", "threads", "posts"}

// Central global information document
type infoDocument struct {
	Document
	DBVersion int `gorethink:"dbVersion"`

	// Is incremented on each new post. Ensures post number uniqueness
	PostCtr uint64 `gorethink:"postCtr"`
}

// Initialize a rethinkDB database
func initRethinkDB() error {
	dbName := config.Config.Rethinkdb.Db
	log.Printf("Initialising database '%s'", dbName)
	if err := DB(r.DBCreate(dbName)).Exec(); err != nil {
		return util.WrapError("Error creating database", err)
	}

	RSession.Use(dbName)

	if err := CreateTables(); err != nil {
		return err
	}

	main := [...]interface{}{
		infoDocument{Document{"info"}, dbVersion, 0},

		// History aka progress counters of boards, that get incremented on
		// post creation
		Document{"histCounts"},
	}
	if err := DB(r.Table("main").Insert(main)).Exec(); err != nil {
		return util.WrapError("Error initializing database", err)
	}

	return CreateIndeces()
}

// CreateTables creates all tables needed for meguca operation
func CreateTables() error {
	for _, table := range AllTables {
		err := DB(r.TableCreate(table)).Exec()
		if err != nil {
			return util.WrapError("Error creating table", err)
		}
	}
	return nil
}

// CreateIndeces create secondary indeces for faster table queries
func CreateIndeces() error {
	err := DB(r.Table("threads").IndexCreate("board")).Exec()
	if err != nil {
		return indexCreationError(err)
	}
	for _, key := range [...]string{"op", "board"} {
		err := DB(r.Table("posts").IndexCreate(key)).Exec()
		if err != nil {
			return indexCreationError(err)
		}
	}

	// Make sure all indeces are ready to avoid the race condition of and index
	// being accessed before its full creation.
	for _, table := range AllTables {
		err := DB(r.Table(table).IndexWait()).Exec()
		if err != nil {
			return util.WrapError("Error waiting for index", err)
		}
	}
	return nil
}

func indexCreationError(err error) error {
	return util.WrapError("Error creating index", err)
}
