/*
 Initialises and loads Redis and RethinkDB
*/

package main

import (
	"fmt"
	r "github.com/dancannon/gorethink"
)

// Shorthand
var db string

const dbVersion = 2

// rSession exports the RethinkDB connection session
var rSession *r.Session

// loadDB establishes connections to RethinkDB and Redis and bootstraps both
// databases, if not yet done.
func loadDB() {
	var err error
	rSession, err = r.Connect(r.ConnectOpts{
		Address: config.Hard.Rethinkdb.Addr,
	})
	throw(err)

	db = config.Hard.Rethinkdb.Db
	var isCreated bool
	rGet(r.DBList().Contains(db)).One(&isCreated)
	if !isCreated {
		initRethinkDB()
	} else {
		rSession.Use(db)
		var version int
		rGet(r.Table("main").Get("info").Field("dbVersion")).One(&version)
		if version != dbVersion {
			panic(fmt.Sprintf("Incompatible RethinkDB database version: %d."+
				"See docs/migration.md", version))
		}
	}
}

// Document is a eneric RethinkDB Document. For DRY-ness.
type Document struct {
	ID string `gorethink:"id"`
}

// ParenthoodCache maps posts to their parent boards and threads
type ParenthoodCache struct {
	OPs    map[string]uint64
	Boards map[string]string `gorethink:"boards"`
}

func initRethinkDB() {
	rExec(r.DBCreate(db))
	rSession.Use(db)
	rExec(r.TableCreate("main"))
	rExec(r.Table("main").Insert([3]interface{}{
		struct {
			Document
			DBVersion int `gorethink:"dbVersion"`

			// Is incremented on each new post. Ensures post number uniqueness
			PostCtr uint64 `gorethink:"postCtr"`
		}{
			Document{"info"}, dbVersion, 0,
		},

		// Contains various board- and post-related statistics
		struct {
			Document
			ParenthoodCache
		}{
			Document{"cache"},
			ParenthoodCache{map[string]uint64{}, map[string]string{}},
		},

		// History aka progress counters of boards, that get incremented on
		// any update
		Document{"histCounts"},
	}))
	rExec(r.TableCreate("threads"))
	rExec(r.Table("threads").IndexCreate("board"))
}
