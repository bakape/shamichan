package db

import (
	"testing"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/imager/assets"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

// Only the database connection and initialisation functions
type DBInit struct{}

var _ = Suite(&DBInit{})

// All other functions that depend on the database
type Tests struct{}

var _ = Suite(&Tests{})

var testDBName string

func (d *Tests) SetUpSuite(c *C) {
	DBName = UniqueDBName()
	c.Assert(Connect(), IsNil)
	c.Assert(InitDB(), IsNil)
	c.Assert(assets.CreateDirs(), IsNil)
}

func (*Tests) SetUpTest(c *C) {
	// Clear all documents from all tables before each test.
	for _, table := range AllTables {
		c.Assert(Write(r.Table(table).Delete()), IsNil)
	}
	config.Set(config.Configs{
		Boards: []string{"a"},
	})
}

func (*Tests) TearDownTest(c *C) {
	c.Assert(assets.ResetDirs(), IsNil)
}

func (d *Tests) TearDownSuite(c *C) {
	c.Assert(r.DBDrop(DBName).Exec(RSession), IsNil)
	c.Assert(RSession.Close(), IsNil)
	c.Assert(assets.DeleteDirs(), IsNil)
}

func (*Tests) TestVerifyVersion(c *C) {
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
		"incompatible RethinkDB database version: 0.*",
	)
}

func (*DBInit) TestLoadDB(c *C) {
	DBName = UniqueDBName()
	isTest = true
	defer func() {
		c.Assert(Write(r.DBDrop(DBName)), IsNil)
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

	indexes := [...]struct {
		table, index string
	}{
		{"threads", "board"},
		{"threads", "post"},
	}
	for _, i := range indexes {
		var hasIndex bool
		err = One(r.Table(i.table).IndexList().Contains(i.index), &hasIndex)
		c.Assert(err, IsNil)
		if !hasIndex {
			c.Fatalf(
				"no secondary index '%s' created for table '%s'",
				i.index,
				i.table,
			)
		}
	}

	var info infoDocument
	c.Assert(One(GetMain("info"), &info), IsNil)
	c.Assert(info, Equals, infoDocument{Document{"info"}, dbVersion, 0})

	var boardCtrs Document
	c.Assert(One(GetMain("boardCtrs"), &boardCtrs), IsNil)
	c.Assert(boardCtrs, Equals, Document{"boardCtrs"})

	var conf config.Configs
	c.Assert(One(GetMain("config"), &conf), IsNil)
	c.Assert(conf, DeepEquals, config.Defaults)

	var exists bool
	c.Assert(One(GetAccount("admin").Eq(nil).Not(), &exists), IsNil)
	c.Assert(exists, Equals, true)

	c.Assert(RSession.Close(), IsNil)
	c.Assert(LoadDB(), IsNil)
}
