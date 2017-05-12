// Initializes and loads RethinkDB

package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"meguca/auth"
	"meguca/config"
	"meguca/util"
	"time"

	"github.com/boltdb/bolt"
	_ "github.com/lib/pq" // Postgres driver
)

const (
	version = 12

	// TestConnArgs contains ConnArgs used for tests
	TestConnArgs = `user=meguca password=meguca dbname=meguca_test sslmode=disable binary_parameters=yes`
)

var (
	// ConnArgs specifies the PostgreSQL connection arguments
	ConnArgs = `user=meguca password=meguca dbname=meguca sslmode=disable binary_parameters=yes`

	// IsTest can be overridden to not launch several infinite loops during tests
	// or check DB version
	IsTest bool

	// Stores the postgres database instance
	db *sql.DB

	// Embedded database for temporary storage
	boltDB *bolt.DB
)

var upgrades = map[uint]func(*sql.Tx) error{
	1: func(tx *sql.Tx) (err error) {
		// Delete legacy thread column
		_, err = tx.Exec(
			`ALTER TABLE threads
				DROP COLUMN locked`,
		)
		if err != nil {
			return
		}

		// Delete legacy board columns
		_, err = tx.Exec(
			`ALTER TABLE boards
				DROP COLUMN hashCommands,
				DROP COLUMN codeTags`,
		)
		return
	},
	2: func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE threads
				DROP COLUMN log`,
		)
		return
	},
	3: func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE boards
				DROP COLUMN ctr`,
		)
		return
	},
	// Restore correct image counters after incorrect updates
	4: func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`UPDATE threads
				SET imageCtr = (SELECT COUNT(*) FROM posts
					WHERE SHA1 IS NOT NULL
						AND op = threads.id
				)`,
		)
		return
	},
	5: func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE images
				ADD COLUMN Title varchar(100) not null default '',
				ADD COLUMN Artist varchar(100) not null default ''`,
		)
		return
	},
	6: func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE posts
				ADD COLUMN sage bool`,
		)
		return
	},
	7: func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(`DROP INDEX deleted`)
		return
	},
	// Set default expiry configs, to keep all threads from deleting
	8: func(tx *sql.Tx) (err error) {
		var s string
		err = tx.QueryRow("SELECT val FROM main WHERE id = 'config'").Scan(&s)
		if err != nil {
			return
		}
		conf, err := decodeConfigs(s)
		if err != nil {
			return
		}

		conf.ThreadExpiryMin = config.Defaults.ThreadExpiryMin
		conf.ThreadExpiryMax = config.Defaults.ThreadExpiryMax
		buf, err := json.Marshal(conf)
		if err != nil {
			return
		}
		_, err = tx.Exec(
			`UPDATE main
				SET val = $1
				WHERE id = 'config'`,
			string(buf),
		)
		return
	},
	9: func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE boards
				ADD COLUMN disableRobots bool not null default false`,
		)
		return
	},
	10: func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE threads
				ADD COLUMN sticky bool default false`,
		)
		return
	},
	11: func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE bans
				ADD COLUMN forPost bigint default 0`,
		)
		return
	},
}

// LoadDB establishes connections to RethinkDB and Redis and bootstraps both
// databases, if not yet done.
func LoadDB() (err error) {
	db, err = sql.Open("postgres", ConnArgs)
	if err != nil {
		return err
	}

	var exists bool
	err = db.QueryRow(getQuery("init/check_db_exists.sql")).Scan(&exists)
	if err != nil {
		return err
	}

	tasks := make([]func() error, 0, 6)
	if !exists {
		tasks = append(tasks, initDB)
	} else if err = checkVersion(); err != nil {
		return
	}
	tasks = append(tasks, genPrepared)
	if !exists {
		tasks = append(tasks, CreateAdminAccount)
	}
	tasks = append(tasks, openBoltDB, loadConfigs, loadBoardConfigs, loadBans)
	if err := util.Waterfall(tasks...); err != nil {
		return err
	}

	if !IsTest {
		go runCleanupTasks()
	}
	return nil
}

// Check database version perform any upgrades
func checkVersion() (err error) {
	var v uint
	err = db.QueryRow(`select val from main where id = 'version'`).Scan(&v)
	if err != nil {
		return
	}

	var tx *sql.Tx
	for i := v; i < version; i++ {
		log.Printf("upgrading database to version %d\n", i+1)
		tx, err = db.Begin()
		if err != nil {
			return
		}

		err = upgrades[i](tx)
		if err != nil {
			return rollBack(tx, err)
		}

		// Write new version number
		_, err = tx.Exec(
			`update main set val = $1 where id = 'version'`,
			i+1,
		)
		if err != nil {
			return rollBack(tx, err)
		}

		err = tx.Commit()
		if err != nil {
			return
		}
	}

	return
}

func rollBack(tx *sql.Tx, err error) error {
	if rbErr := tx.Rollback(); rbErr != nil {
		err = util.WrapError(err.Error(), rbErr)
	}
	return err
}

func openBoltDB() (err error) {
	boltDB, err = bolt.Open("db.db", 0600, &bolt.Options{
		Timeout: time.Second,
	})
	if err != nil {
		return
	}
	return boltDB.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("open_bodies"))
		return err
	})
}

// initDB initializes a database
func initDB() error {
	log.Println("initializing database")

	conf, err := json.Marshal(config.Defaults)
	if err != nil {
		return err
	}

	q := fmt.Sprintf(getQuery("init/init.sql"), version, string(conf))
	_, err = db.Exec(q)
	return err
}

// CreateAdminAccount writes a fresh admin account with the default password to
// the database
func CreateAdminAccount() error {
	hash, err := auth.BcryptHash("password", 10)
	if err != nil {
		return err
	}
	return RegisterAccount("admin", hash)
}

// ClearTables deletes the contents of specified DB tables. Only used for tests.
func ClearTables(tables ...string) error {
	for _, t := range tables {
		// Clear open post body bucket
		switch t {
		case "boards", "threads", "posts":
			err := boltDB.Update(func(tx *bolt.Tx) error {
				buc := tx.Bucket([]byte("open_bodies"))
				c := buc.Cursor()
				for k, _ := c.First(); k != nil; k, _ = c.Next() {
					err := buc.Delete(k)
					if err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				return err
			}
		}

		if _, err := db.Exec(`DELETE FROM ` + t); err != nil {
			return err
		}
	}
	return nil
}
