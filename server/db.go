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

// Shorthand
var db string

const dbVersion = 2

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

	db = config.Config.Hard.Rethinkdb.Db
	var isCreated bool
	Get(r.DBList().Contains(db)).One(&isCreated)
	if !isCreated {
		initRethinkDB()
	} else {
		Session.Use(db)
		var version int
		Get(r.Table("main").Get("info").Field("dbVersion")).One(&version)
		verifyVersion(version, "RethinkDB")
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

// Generic RethinkDB document
type document struct {
	ID string `gorethink:"id"`
}

type stringMap map[string]string
type intMap map[string]int

// Maps posts to their parent boards and threads
type parenthoodCache struct {
	OPs    intMap
	Boards stringMap `gorethink:"boards"`
}

func initRethinkDB() {
	Exec(r.DBCreate(db))
	Session.Use(db)
	Exec(r.TableCreate("main"))
	Exec(r.Table("main").Insert([3]interface{}{
		struct {
			document
			DBVersion int `gorethink:"dbVersion"`

			// Is incremented on each new post. Ensures post number uniqueness
			PostCounter int `gorethink:"postCounter"`
		}{
			document{"info"}, dbVersion, 0,
		},

		// Contains various board- and post-related statistics
		struct {
			document
			parenthoodCache
		}{
			document{"cache"}, parenthoodCache{intMap{}, stringMap{}},
		},

		// History aka progress counters of boards, that get incremented on
		// any update
		document{"histCounts"},
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
