// Package db initialises and loads Redis and RethinkDB
package db

import (
	"fmt"
	r "github.com/dancannon/gorethink"
	"gopkg.in/redis.v3"
	"meguca/config"
	"meguca/util"
	"strconv"
)

const dbVersion = 2

var throw = util.Throw

var db string

// RedisClient creates a new redis client
func RedisClient() *redis.Client {
	var conf = config.Config.Hard.Redis
	return redis.NewClient(&redis.Options{
		Addr: conf.Addr,
		DB:   conf.Db,
	})
}

// Redis exports the main redis client
var Redis *redis.Client

// Session exports the RethinkDB connection session
var Session *r.Session

// Info stores the central gloabal information and stats
type Info struct {
	ID        string `gorethink:"id"`
	DBVersion int    `gorethink:"dbVersion"`
	PostCtr   int    `gorethink:"postCtr"`
}

// Load establishes connections to RethinkDB and Redis and bootstraps both
// databases, if not yet done.
func Load() {
	loadRethinkDB()
	loadRedis()
}

func loadRethinkDB() {
	var err error
	Session, err = r.Connect(r.ConnectOpts{
		Address: config.Config.Hard.Rethinkdb.Addr,
	})
	throw(err)

	// Shorthand
	db = config.Config.Hard.Rethinkdb.Db
	var res bool
	Get(r.DBList().Contains(db)).One(&res)
	if !res {
		initRethinkDB()
	} else {
		Session.Use(db)
		var res Info
		Get(r.Table("main").Get("info")).One(&res)
		verifyVersion(res.DBVersion, "RethinkDB")
	}
}

func loadRedis() {
	Redis = RedisClient()
	if redisVersion, err := Redis.Get("dbVersion").Result(); err == redis.Nil {
		// Fresh database. Write version number.
		throw(Redis.Set("dbVersion", dbVersion, 0).Err())
	} else {
		throw(err)

		// Verify Redis database version
		conv, err1 := strconv.ParseInt(redisVersion, 10, 64)
		throw(err1)
		verifyVersion(int(conv), "Redis")
	}
}

// BoardCounters stores history counters of boards. Used for building etags.
type BoardCounters struct {
	ID     string         `gorethink:"id"`
	Boards map[string]int `gorethink:"boards"`
}

func initRethinkDB() {
	Run(r.DBCreate(db))
	Session.Use(db)
	Run(r.TableCreate("main"))
	Run(r.Table("main").Insert([2]interface{}{
		Info{"info", dbVersion, 0},
		BoardCounters{"boardCtrs", map[string]int{}},
	}))
	Run(r.TableCreate("threads"))
	Run(r.Table("threads").IndexCreate("board"))
}

func verifyVersion(version int, dbms string) {
	if version != dbVersion {
		panic(fmt.Sprintf("Incompatible %v database version: %d."+
			"See docs/migration.md", dbms, version))
	}
}

// Get is a shorthand for executing RethinkDB queries and panicing on error.
func Get(query r.Term) *r.Cursor {
	cursor, err := query.Run(Session)
	throw(err)
	return cursor
}

// Run executes a RethinkDB query and panics on error. To be used, when the
// returned status is unneeded and we want the goroutine to crash on error.
func Run(query r.Term) {
	cursor, err := query.Run(Session)
	throw(err)
	cursor.Close()
}
