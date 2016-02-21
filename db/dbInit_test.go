package db

import (
	"errors"
	"github.com/bakape/meguca/config"
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
type DBSuite struct {
	dbName string
}

var _ = Suite(&DBSuite{})

var testDBName string

func (d *DBSuite) SetUpSuite(c *C) {
	d.dbName = uniqueDBName()
	connectToRethinkDb(c)
	DB()(r.DBCreate(d.dbName)).Exec()
	RSession.Use(d.dbName)
	CreateTables()
	CreateIndeces()
}

// Returns a unique datatabase name. Needed so multiple concurent `go test`
// don't clash in the same database.
func uniqueDBName() string {
	return "meguca_tests_" + strconv.FormatInt(time.Now().UnixNano(), 10)
}

func connectToRethinkDb(c *C) {
	var err error
	RSession, err = r.Connect(r.ConnectOpts{
		Address: "localhost:28015",
	})
	c.Assert(err, IsNil)
}

func (*DBSuite) SetUpTest(_ *C) {
	config.Config = config.Server{}
	config.Config.Boards.Enabled = []string{"a"}
}

// Clear all documents from all tables after each test.
func (*DBSuite) TearDownTest(_ *C) {
	for _, table := range AllTables {
		DB()(r.Table(table).Delete()).Exec()
	}
}

func (d *DBSuite) TearDownSuite(c *C) {
	c.Assert(r.DBDrop(d.dbName).Exec(RSession), IsNil)
	c.Assert(RSession.Close(), IsNil)
}

func (*DBSuite) TestVerifyVersion(c *C) {
	// Correct DB version
	DB()(r.Table("main").Insert(map[string]interface{}{
		"id":        "info",
		"dbVersion": dbVersion,
	})).Exec()
	verifyDBVersion()

	// Incompatible DB version
	DB()(r.Table("main").Get("info").Update(map[string]int{
		"dbVersion": 0,
	})).Exec()
	err := errors.New("Incompatible RethinkDB database version: 0." +
		"See docs/migration.md")
	c.Assert(verifyDBVersion, Panics, err)
}

func (*DBInit) TestDb(c *C) {
	query := r.Table("posts").Get(1)
	standard := DatabaseHelper{query}
	c.Assert(DB()(query), DeepEquals, standard)
}

func (*DBInit) TestLoadDB(c *C) {
	config.Config = config.Server{}
	config.Config.Rethinkdb.Addr = "localhost:28015"
	dbName := uniqueDBName()
	config.Config.Rethinkdb.Db = dbName
	defer func() {
		DB()(r.DBDrop(dbName)).Exec()
		c.Assert(RSession.Close(), IsNil)
	}()
	LoadDB()

	var missingTables []string
	DB()(r.Expr(AllTables).Difference(r.TableList())).One(&missingTables)
	for _, table := range missingTables {
		c.Fatalf("table '%s' not created", table)
	}

	var hasIndex bool
	DB()(r.Table("threads").IndexList().Contains("board")).One(&hasIndex)
	if !hasIndex {
		indexMissing("threads", "board", c)
	}

	for _, table := range [...]string{"posts", "updates"} {
		var missingIndeces []string
		DB()(r.
			Expr([...]string{"op", "board"}).
			Difference(r.Table(table).IndexList()),
		).One(&missingIndeces)
		for _, index := range missingIndeces {
			indexMissing(table, index, c)
		}
	}

	var info infoDocument
	DB()(r.Table("main").Get("info")).One(&info)
	c.Assert(info, Equals, infoDocument{Document{"info"}, dbVersion, 0})

	var histCounts Document
	DB()(r.Table("main").Get("histCounts")).One(&histCounts)
	c.Assert(histCounts, Equals, Document{"histCounts"})

	c.Assert(RSession.Close(), IsNil)
	LoadDB()
}

func indexMissing(table, index string, c *C) {
	c.Fatalf("no secondary index '%s' created for table '%s'", index, table)
}
