package imager

import (
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

type DB struct {
	dbName string
}

var _ = Suite(&DB{})

func (d *DB) SetUpSuite(c *C) {
	d.dbName = db.UniqueDBName()
	c.Assert(db.Connect(""), IsNil)
	c.Assert(db.InitDB(d.dbName), IsNil)
}

func (d *DB) TearDownSuite(c *C) {
	c.Assert(db.DB(r.DBDrop(d.dbName)).Exec(), IsNil)
	c.Assert(db.RSession.Close(), IsNil)
}

func (d *DB) TearDownTest(c *C) {
	query := db.GetMain("imageHashes").Replace(db.Document{"imageHashes"})
	c.Assert(db.DB(query).Exec(), IsNil)
}

func (*DB) TestCleanUpHashes(c *C) {
	expired := HashEntry{
		ID:   20,
		Hash: 222,
	}
	fresh := HashEntry{
		ID:   50,
		Hash: 555,
	}
	update := map[string][]DatabaseHashEntry{
		"hashes": []DatabaseHashEntry{
			{
				HashEntry: expired,
				Expires:   r.Now().Sub(3 * 60 * 60),
			},
			{
				HashEntry: fresh,
				Expires:   r.Now(),
			},
		},
	}
	c.Assert(db.DB(db.GetMain("imageHashes").Update(update)).Exec(), IsNil)

	conf := config.ServerConfigs{}
	conf.Images.DulicateLifetime = 2 * 60 * 60
	config.Set(conf)

	c.Assert(cleanUpHashes(), IsNil)
	var hashes []HashEntry
	query := db.GetMain("imageHashes").Field("hashes")
	c.Assert(db.DB(query).All(&hashes), IsNil)
	c.Assert(hashes, DeepEquals, []HashEntry{fresh})
}
