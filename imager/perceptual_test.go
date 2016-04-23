package imager

import (
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
	"time"
)

type DB struct {
	dbName           string
	perceptualCloser chan struct{}
}

var _ = Suite(&DB{})

func (d *DB) SetUpSuite(c *C) {
	d.dbName = db.UniqueDBName()
	c.Assert(db.Connect(""), IsNil)
	c.Assert(db.InitDB(d.dbName), IsNil)
}

func (d *DB) SetUpTest(c *C) {
	query := db.GetMain("imageHashes").Replace(map[string]interface{}{
		"id":     "imageHashes",
		"hashes": []string{},
	})
	c.Assert(db.DB(query).Exec(), IsNil)

	conf := config.ServerConfigs{}
	conf.Images.Max.Height = 10000
	conf.Images.Max.Width = 10000
	conf.Images.DuplicateThreshold = 1
	config.Set(conf)

	cleanUpInterval = time.Second * 3
	d.perceptualCloser = make(chan struct{})
	go handlePerceptualHashes(d.perceptualCloser)
}

func (d *DB) TearDownSuite(c *C) {
	c.Assert(db.DB(r.DBDrop(d.dbName)).Exec(), IsNil)
	c.Assert(db.RSession.Close(), IsNil)
}

func (d *DB) TearDownTest(c *C) {
	cleanUpInterval = time.Minute
	close(d.perceptualCloser)
}

func (*DB) TestCleanUpHashes(c *C) {
	expired := hashEntry{
		ID:   20,
		Hash: 222,
	}
	fresh := hashEntry{
		ID:   50,
		Hash: 555,
	}
	update := map[string][]databaseHashEntry{
		"hashes": []databaseHashEntry{
			{
				hashEntry: expired,
				Expires:   r.Now().Sub(3 * 60 * 60),
			},
			{
				hashEntry: fresh,
				Expires:   r.Now(),
			},
		},
	}
	c.Assert(db.DB(db.GetMain("imageHashes").Update(update)).Exec(), IsNil)

	conf := config.ServerConfigs{}
	conf.Images.DulicateLifetime = 2 * 60 * 60
	config.Set(conf)

	// Wait for the 3 second cleanUp timer to kick in
	time.Sleep(time.Second * 5)
	var hashes []hashEntry
	query := db.GetMain("imageHashes").Field("hashes")
	c.Assert(db.DB(query).All(&hashes), IsNil)
	c.Assert(hashes, DeepEquals, []hashEntry{fresh})
}

func (*DB) TestFreshHashAdding(c *C) {
	std := hashEntry{
		ID:   1,
		Hash: 111,
	}
	res := make(chan uint64)
	dedupImage <- dedupRequest{
		entry: std,
		res:   res,
	}

	c.Assert(<-res, Equals, uint64(0))
	query := db.GetMain("imageHashes").Field("hashes")
	var hashes []hashEntry
	c.Assert(db.DB(query).All(&hashes), IsNil)
	c.Assert(hashes, DeepEquals, []hashEntry{std})
}

func closeHandler(close chan<- struct{}) {
	close <- struct{}{}
}

func (*DB) TestDuplicateMatching(c *C) {
	base := hashEntry{
		ID:   1,
		Hash: 7,
	}
	noMatch := hashEntry{
		ID:   2,
		Hash: 1,
	}
	match := hashEntry{
		ID:   3,
		Hash: 3,
	}
	c.Assert(persistHash(base), IsNil)

	res := make(chan uint64)
	dedupImage <- dedupRequest{
		entry: match,
		res:   res,
	}
	c.Assert(<-res, Equals, uint64(1))

	dedupImage <- dedupRequest{
		entry: noMatch,
		res:   res,
	}
	c.Assert(<-res, Equals, uint64(0))
}
