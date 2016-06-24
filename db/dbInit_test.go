package db

import (
	"testing"

	"github.com/bakape/meguca/config"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
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
	d.dbName = UniqueDBName()
	c.Assert(Connect(""), IsNil)
	c.Assert(InitDB(d.dbName), IsNil)
}

func (*DBSuite) SetUpTest(c *C) {
	// Clear all documents from all tables after each test.
	for _, table := range AllTables {
		c.Assert(Write(r.Table(table).Delete()), IsNil)
	}

	conf := config.ServerConfigs{}
	conf.Boards.Enabled = []string{"a"}
	config.Set(conf)
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
	c.Assert(Write(r.Table("main").Insert(info)), IsNil)
	c.Assert(verifyDBVersion(), IsNil)

	// Incompatible DB version
	update := map[string]int{"dbVersion": 0}
	c.Assert(Write(GetMain("info").Update(update)), IsNil)
	c.Assert(
		verifyDBVersion(),
		ErrorMatches,
		"Incompatible RethinkDB database version: 0.*",
	)
}

func (*DBInit) TestLoadDB(c *C) {
	conf := config.ServerConfigs{}
	conf.Rethinkdb.Addr = "localhost:28015"
	dbName := UniqueDBName()
	conf.Rethinkdb.Db = dbName
	config.Set(conf)
	defer func() {
		c.Assert(Write(r.DBDrop(dbName)), IsNil)
		c.Assert(RSession.Close(), IsNil)
	}()
	c.Assert(LoadDB(), IsNil)

	var missingTables []string
	query := r.Expr(AllTables).Difference(r.TableList()).Default([]string{})
	err := All(query, &missingTables)
	c.Assert(err, IsNil)
	for _, table := range missingTables {
		c.Fatalf("table '%s' not created", table)
	}

	indexes := map[string]string{
		"threads": "board",
	}
	for table, index := range indexes {
		var hasIndex bool
		err = One(r.Table(table).IndexList().Contains(index), &hasIndex)
		c.Assert(err, IsNil)
		if !hasIndex {
			c.Fatalf(
				"no secondary index '%s' created for table '%s'",
				index,
				table,
			)
		}
	}

	var info infoDocument
	c.Assert(One(GetMain("info"), &info), IsNil)
	c.Assert(info, Equals, infoDocument{Document{"info"}, dbVersion, 0})

	var histCounts Document
	c.Assert(One(GetMain("histCounts"), &histCounts), IsNil)
	c.Assert(histCounts, Equals, Document{"histCounts"})

	c.Assert(RSession.Close(), IsNil)
	c.Assert(LoadDB(), IsNil)
}
