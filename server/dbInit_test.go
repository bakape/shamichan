package server

import (
	"errors"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
	"strconv"
	"testing"
	"time"
)

func Test(t *testing.T) { TestingT(t) }

// Only the database connection and initialisation functions
type DBInit struct{}

var _ = Suite(&DBInit{})

// All other functions that depend on the database
type DB struct {
	dbName string
}

var _ = Suite(&DB{})

var testDBName string

func (d *DB) SetUpSuite(c *C) {
	d.dbName = uniqueDBName()
	connectToRethinkDb(c)
	db()(r.DBCreate(d.dbName)).Exec()
	rSession.Use(d.dbName)
	createTables()
	createIndeces()
}

// Returns a unique datatabase name. Needed so multiple concurent `go test`
// don't clashin the same database.
func uniqueDBName() string {
	return "meguca_tests_" + strconv.FormatInt(time.Now().UnixNano(), 10)
}

func connectToRethinkDb(c *C) {
	var err error
	rSession, err = r.Connect(r.ConnectOpts{
		Address: "localhost:28015",
	})
	c.Assert(err, IsNil)
}

// Clear all documents from all tables after each test.
func (*DB) TearDownTest(c *C) {
	for _, table := range allTables {
		db()(r.Table(table).Delete()).Exec()
	}
}

func (d *DB) TearDownSuite(c *C) {
	c.Assert(r.DBDrop(d.dbName).Exec(rSession), IsNil)
	c.Assert(rSession.Close(), IsNil)
}

func (*DB) TestVerifyVersion(c *C) {
	// Correct DB version
	db()(r.Table("main").Insert(map[string]interface{}{
		"id":        "info",
		"dbVersion": dbVersion,
	})).Exec()
	verifyDBVersion()

	// Incompatible DB version
	db()(r.Table("main").Get("info").Update(map[string]int{
		"dbVersion": 0,
	})).Exec()
	err := errors.New("Incompatible RethinkDB database version: 0." +
		"See docs/migration.md")
	c.Assert(verifyDBVersion, Panics, err)
}

func (*DBInit) TestDb(c *C) {
	query := r.Table("posts").Get(1)
	standard := DatabaseHelper{query}
	c.Assert(db()(query), DeepEquals, standard)
}

func (*DBInit) TestLoadDB(c *C) {
	config = serverConfigs{}
	config.Rethinkdb.Addr = "localhost:28015"
	dbName := uniqueDBName()
	config.Rethinkdb.Db = dbName
	defer func() {
		db()(r.DBDrop(dbName)).Exec()
		c.Assert(rSession.Close(), IsNil)
	}()
	loadDB()

	var missingTables []string
	db()(r.Expr(allTables).Difference(r.TableList())).One(&missingTables)
	for _, table := range missingTables {
		c.Fatalf("table '%s' not created", table)
	}

	var hasIndex bool
	db()(r.Table("threads").IndexList().Contains("board")).One(&hasIndex)
	if !hasIndex {
		indexMissing("threads", "board", c)
	}

	for _, table := range [...]string{"posts", "updates"} {
		var missingIndeces []string
		db()(r.
			Expr([...]string{"op", "board"}).
			Difference(r.Table(table).IndexList()),
		).One(&missingIndeces)
		for _, index := range missingIndeces {
			indexMissing(table, index, c)
		}
	}

	var info infoDocument
	db()(r.Table("main").Get("info")).One(&info)
	c.Assert(info, Equals, infoDocument{Document{"info"}, dbVersion, 0})

	var histCounts Document
	db()(r.Table("main").Get("histCounts")).One(&histCounts)
	c.Assert(histCounts, Equals, Document{"histCounts"})

	c.Assert(rSession.Close(), IsNil)
	loadDB()
}

func indexMissing(table, index string, c *C) {
	c.Fatalf("no secondary index '%s' created for table '%s'", index, table)
}
