package imager

import (
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
	"time"
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
	cleanUpInterval = time.Minute
	query := db.GetMain("imageHashes").Replace(map[string]interface{}{
		"id":     "imageHashes",
		"hashes": []string{},
	})
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

	cleanUpInterval = time.Second * 3
	close := make(chan struct{})
	go handlePerceptualHashes(close)
	defer closeHandler(close)

	time.Sleep(time.Second * 5)
	var hashes []HashEntry
	query := db.GetMain("imageHashes").Field("hashes")
	c.Assert(db.DB(query).All(&hashes), IsNil)
	c.Assert(hashes, DeepEquals, []HashEntry{fresh})
}

func (*DB) TestFreshHashAdding(c *C) {
	close := make(chan struct{})
	go handlePerceptualHashes(close)
	defer closeHandler(close)

	std := HashEntry{
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
	var hashes []HashEntry
	c.Assert(db.DB(query).All(&hashes), IsNil)
	c.Assert(hashes, DeepEquals, []HashEntry{std})
}

func closeHandler(close chan<- struct{}) {
	close <- struct{}{}
}

func (*DB) TestDuplicateMatching(c *C) {
	close := make(chan struct{})
	go handlePerceptualHashes(close)
	defer closeHandler(close)

	conf := config.ServerConfigs{}
	conf.Images.DuplicateThreshold = 1
	config.Set(conf)

	base := HashEntry{
		ID:   1,
		Hash: 7,
	}
	noMatch := HashEntry{
		ID:   2,
		Hash: 1,
	}
	match := HashEntry{
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
