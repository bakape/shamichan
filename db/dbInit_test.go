package db

import (
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
	c.Assert(DB()(r.DBCreate(d.dbName)).Exec(), IsNil)
	RSession.Use(d.dbName)
	c.Assert(CreateTables(), IsNil)
	c.Assert(CreateIndeces(), IsNil)
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
func (*DBSuite) TearDownTest(c *C) {
	for _, table := range AllTables {
		c.Assert(DB()(r.Table(table).Delete()).Exec(), IsNil)
	}
}

func (d *DBSuite) TearDownSuite(c *C) {
	c.Assert(r.DBDrop(d.dbName).Exec(RSession), IsNil)
	c.Assert(RSession.Close(), IsNil)
}

func (*DBSuite) TestVerifyVersion(c *C) {
	// Correct DB version
	info := map[string]interface{}{
		"id":        "info",
		"dbVersion": dbVersion,
	}
	c.Assert(DB()(r.Table("main").Insert(info)).Exec(), IsNil)
	c.Assert(verifyDBVersion(), IsNil)

	// Incompatible DB version
	update := map[string]int{"dbVersion": 0}
	c.Assert(DB()(r.Table("main").Get("info").Update(update)).Exec(), IsNil)
	c.Assert(
		verifyDBVersion(),
		ErrorMatches,
		"Incompatible RethinkDB database version: 0.*",
	)
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
		c.Assert(DB()(r.DBDrop(dbName)).Exec(), IsNil)
		c.Assert(RSession.Close(), IsNil)
	}()
	c.Assert(LoadDB(), IsNil)

	var missingTables []string
	err := DB()(r.Expr(AllTables).Difference(r.TableList())).One(&missingTables)
	c.Assert(err, IsNil)
	for _, table := range missingTables {
		c.Fatalf("table '%s' not created", table)
	}

	var hasIndex bool
	err = DB()(r.Table("threads").IndexList().Contains("board")).One(&hasIndex)
	c.Assert(err, IsNil)
	if !hasIndex {
		indexMissing("threads", "board", c)
	}

	var missingIndeces []string
	query := r.
		Expr([...]string{"op", "board"}).
		Difference(r.Table("posts").IndexList())
	c.Assert(DB()(query).One(&missingIndeces), IsNil)
	for _, index := range missingIndeces {
		indexMissing("posts", index, c)
	}

	var info infoDocument
	c.Assert(DB()(r.Table("main").Get("info")).One(&info), IsNil)
	c.Assert(info, Equals, infoDocument{Document{"info"}, dbVersion, 0})

	var histCounts Document
	c.Assert(DB()(r.Table("main").Get("histCounts")).One(&histCounts), IsNil)
	c.Assert(histCounts, Equals, Document{"histCounts"})

	c.Assert(RSession.Close(), IsNil)
	c.Assert(LoadDB(), IsNil)
}

func indexMissing(table, index string, c *C) {
	c.Fatalf("no secondary index '%s' created for table '%s'", index, table)
}
