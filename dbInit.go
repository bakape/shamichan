/*
 Initialises and loads Redis and RethinkDB
*/

package main

import (
	"fmt"
	r "github.com/dancannon/gorethink"
	"gopkg.in/redis.v3"
	"strconv"
)

// Shorthand
var db string

const dbVersion = 2

// newRedisClient creates a new redis client
func newRedisClient() *redis.Client {
	var conf = config.Hard.Redis
	return redis.NewClient(&redis.Options{
		Addr: conf.Addr,
		DB:   conf.Db,
	})
}

// redis stores exports the main redis client
var redisClient *redis.Client

// rSession exports the RethinkDB connection session
var rSession *r.Session

// loadDB establishes connections to RethinkDB and Redis and bootstraps both
// databases, if not yet done.
func loadDB() {
	loadRethinkDB()
	loadRedis()
}

func loadRethinkDB() {
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
		verifyVersion(version, "RethinkDB")
	}
}

func loadRedis() {
	redisClient = newRedisClient()
	redisVersion, err := redisClient.Get("dbVersion").Result()
	if err == redis.Nil {
		// Fresh database. Write version number.
		throw(redisClient.Set("dbVersion", dbVersion, 0).Err())
	} else {
		throw(err)

		// Verify Redis database version
		conv, err1 := strconv.ParseInt(redisVersion, 10, 64)
		throw(err1)
		verifyVersion(int(conv), "Redis")
	}
}

// Document is a eneric RethinkDB Document. For DRY-ness.
type Document struct {
	ID string `gorethink:"id"`
}

type stringMap map[string]string
type intMap map[string]int

// ParenthoodCache maps posts to their parent boards and threads
type ParenthoodCache struct {
	OPs    intMap
	Boards stringMap `gorethink:"boards"`
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
			PostCounter int `gorethink:"postCounter"`
		}{
			Document{"info"}, dbVersion, 0,
		},

		// Contains various board- and post-related statistics
		struct {
			Document
			ParenthoodCache
		}{
			Document{"cache"}, ParenthoodCache{intMap{}, stringMap{}},
		},

		// History aka progress counters of boards, that get incremented on
		// any update
		Document{"histCounts"},
	}))
	rExec(r.TableCreate("threads"))
	rExec(r.Table("threads").IndexCreate("board"))
}

func verifyVersion(version int, dbms string) {
	if version != dbVersion {
		panic(fmt.Sprintf("Incompatible %v database version: %d."+
			"See docs/migration.md", dbms, version))
	}
}

// rGet is a shorthand for executing RethinkDB queries and panicing on error.
func rGet(query r.Term) *r.Cursor {
	cursor, err := query.Run(rSession)
	throw(err)
	return cursor
}

// rExec executes a RethinkDB query and panics on error. To be used, when the
// returned status is unneeded and we want the goroutine to crash on error.
func rExec(query r.Term) {
	throw(query.Exec(rSession))
}
