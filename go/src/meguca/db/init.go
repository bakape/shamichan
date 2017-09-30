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
	// TestConnArgs contains ConnArgs used for tests
	TestConnArgs = `user=meguca password=meguca dbname=meguca_test sslmode=disable binary_parameters=yes`
)

var (
	version = len(upgrades) + 1

	// ConnArgs specifies the PostgreSQL connection arguments
	ConnArgs = `user=meguca password=meguca dbname=meguca sslmode=disable binary_parameters=yes`

	// IsTest can be overridden to not launch several infinite loops during
	// tests
	IsTest bool

	// Stores the postgres database instance
	db *sql.DB

	// Embedded database for temporary storage
	boltDB *bolt.DB
)

var upgrades = []func(*sql.Tx) error{
	func(tx *sql.Tx) (err error) {
		// Delete legacy columns
		return execAll(tx,
			`ALTER TABLE threads
				DROP COLUMN locked`,
			`ALTER TABLE boards
				DROP COLUMN hashCommands,
				DROP COLUMN codeTags`,
		)
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE threads
				DROP COLUMN log`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE boards
				DROP COLUMN ctr`,
		)
		return
	},
	// Restore correct image counters after incorrect updates
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`UPDATE threads
				SET imageCtr = (SELECT COUNT(*) FROM posts
					WHERE SHA1 IS NOT NULL
						AND op = threads.id
				)`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE images
				ADD COLUMN Title varchar(100) not null default '',
				ADD COLUMN Artist varchar(100) not null default ''`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE posts
				ADD COLUMN sage bool`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(`DROP INDEX deleted`)
		return
	},
	// Set default expiry configs, to keep all threads from deleting
	func(tx *sql.Tx) (err error) {
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
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE boards
				ADD COLUMN disableRobots bool default false`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE threads
				ADD COLUMN sticky bool default false`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE bans
				ADD COLUMN forPost bigint default 0`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		return execAll(tx,
			`create table mod_log (
				type smallint not null,
				board varchar(3) not null,
				id bigint not null,
				by varchar(20) not null,
				created timestamp default (now() at time zone 'utc')
			)`,
			`create index mod_log_board on mod_log (board)`,
			`create index mod_log_created on mod_log (created)`,
		)
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(`create index sticky on threads (sticky)`)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE posts
				DROP COLUMN backlinks`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`create table banners (
				board varchar(3) not null references boards on delete cascade,
				id smallint not null,
				data bytea not null,
				mime text not null
			);`,
		)
		return
	},
	func(tx *sql.Tx) error {
		return execAll(tx,
			`alter table boards
				alter column id type text`,
			`alter table bans
				alter column board type text`,
			`alter table mod_log
				alter column board type text`,
			`alter table staff
				alter column board type text`,
			`alter table banners
				alter column board type text`,
			`alter table threads
				alter column board type text`,
			`alter table posts
				alter column board type text`,
		)
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`create table loading_animations (
				board text primary key references boards on delete cascade,
				data bytea not null,
				mime text not null
			);`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE boards
				ADD COLUMN defaultCSS text default 'moe'`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE posts
				ADD COLUMN flag char(2)`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE boards
				ADD COLUMN flags bool default false`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`alter table images
				alter column title type varchar(200)`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE boards
				ADD COLUMN NSFW bool default false`,
		)
		return
	},
	func(tx *sql.Tx) error {
		return execAll(tx,
			`create table reports (
				id bigserial primary key,
				target bigint not null,
				board text not null,
				reason text not null,
				by inet not null,
				illegal boolean not null,
				created timestamp default (now() at time zone 'utc')
			)`,
			`create index report_board on reports (board)`,
			`create index report_created on reports (created)`,
		)
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE boards
				ADD COLUMN nonLive bool default false`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE threads
				ADD COLUMN nonLive bool default false`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE boards
				ADD COLUMN posterIDs bool default false`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE posts
				ADD COLUMN posterID text`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE threads
				ADD COLUMN locked bool default false`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE boards
				ADD COLUMN js varchar(5000) default ''`,
		)
		return
	},
}

// LoadDB establishes connections to RethinkDB and Redis and bootstraps both
// databases, if not yet done.
func LoadDB() (err error) {
	db, err = sql.Open("postgres", ConnArgs)
	if err != nil {
		return
	}

	var exists bool
	err = db.QueryRow(getQuery("init/check_db_exists.sql")).Scan(&exists)
	if err != nil {
		return
	}

	tasks := make([]func() error, 0, 16)
	if !exists {
		tasks = append(tasks, initDB)
	} else if err = checkVersion(); err != nil {
		return
	}
	tasks = append(tasks, genPrepared)

	// Run these is parallel
	tasks = append(
		tasks,
		func() error {
			tasks := []func() error{
				openBoltDB, loadConfigs, loadBoardConfigs, loadBans,
				loadBanners, loadLoadingAnimations,
			}
			if !exists {
				tasks = append(tasks, CreateAdminAccount)
			}
			return util.Parallel(tasks...)
		},
	)

	err = util.Waterfall(tasks...)
	if err != nil {
		return
	}

	if !IsTest {
		go runCleanupTasks()
	}
	return nil
}

// Check database version perform any upgrades
func checkVersion() (err error) {
	var v int
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

		err = upgrades[i-1](tx)
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
