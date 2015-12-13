// Initialises and loads Redis and RethinkDB

package server

import (
	"fmt"
	r "github.com/dancannon/gorethink"
	"gopkg.in/redis.v3"
	"meguca/config"
	. "meguca/util"
	"strconv"
)

const dbVersion = 2

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

// Info stores the central global meta information and stats
type Info struct {
	ID        string `gorethink:"id"`
	DBVersion int    `gorethink:"dbVersion"`
}

// LoadDB establishes connections to RethinkDB and Redis and bootstraps both
// databases, if not yet done.
func LoadDB() {
	loadRethinkDB()
	loadRedis()
}

func loadRethinkDB() {
	var err error
	Session, err = r.Connect(r.ConnectOpts{
		Address: config.Config.Hard.Rethinkdb.Addr,
	})
	Throw(err)

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
		Throw(Redis.Set("dbVersion", dbVersion, 0).Err())
	} else {
		Throw(err)

		// Verify Redis database version
		conv, err1 := strconv.ParseInt(redisVersion, 10, 64)
		Throw(err1)
		verifyVersion(int(conv), "Redis")
	}
}

// Cache contains various board- and post-related statistics
type Cache struct {
	ID string `gorethink:"id"`

	// Is incremented on each new post. Ensures post number uniqueness
	PostCounter int `gorethink:"postCounter"`

	// History aka progress counters of boards, that get incremented on any
	// update
	HistoryCounters intMap `gorethink:"historyCounters"`

	// Maps post numbers to their parent threads
	OPs intMap

	// Maps post numbers to their parent boards
	Boards stringMap `gorethink:"boards"`
}

type stringMap map[string]string
type intMap map[string]int

func initRethinkDB() {
	Exec(r.DBCreate(db))
	Session.Use(db)
	Exec(r.TableCreate("main"))
	Exec(r.Table("main").Insert([2]interface{}{
		Info{"info", dbVersion},
		Cache{"cache", 0, intMap{}, intMap{}, stringMap{}},
	}))
	Exec(r.TableCreate("threads"))
	Exec(r.Table("threads").IndexCreate("board"))
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
	Throw(err)
	return cursor
}

// Exec executes a RethinkDB query and panics on error. To be used, when the
// returned status is unneeded and we want the goroutine to crash on error.
func Exec(query r.Term) {
	Throw(query.Exec(Session))
}
