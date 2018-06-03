package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"meguca/auth"
	"meguca/config"
	"meguca/util"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/boltdb/bolt"
	"github.com/go-playground/log"
	_ "github.com/lib/pq" // Postgres driver
)

const (
	// TestConnArgs contains ConnArgs used for tests
	TestConnArgs = `user=meguca password=meguca dbname=meguca_test sslmode=disable binary_parameters=yes`
)

var (
	// ConnArgs specifies the PostgreSQL connection arguments
	ConnArgs = `user=meguca password=meguca dbname=meguca sslmode=disable binary_parameters=yes`

	// IsTest can be overridden to not launch several infinite loops during
	// tests
	IsTest bool

	// Stores the postgres database instance
	db *sql.DB

	// Statement builder and cacher
	sq squirrel.StatementBuilderType

	// Embedded database for temporary storage
	boltDB *bolt.DB
)

// LoadDB establishes connections to RethinkDB and Redis and bootstraps both
// databases, if not yet done.
func LoadDB() (err error) {
	db, err = sql.Open("postgres", ConnArgs)
	if err != nil {
		return
	}

	sq = squirrel.StatementBuilder.
		RunWith(squirrel.NewStmtCacheProxy(db)).
		PlaceholderFormat(squirrel.Dollar)

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
				openBoltDB, loadConfigs, loadBans,
				loadBanners, loadLoadingAnimations,
			}
			if !exists {
				tasks = append(tasks, CreateAdminAccount, createSystemAccount)
			}
			if err := util.Parallel(tasks...); err != nil {
				return err
			}

			// Depends on loadBanners and loadLoadingAnimations, so has to be
			// sequential
			return loadBoardConfigs()
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
		log.Infof("upgrading database to version %d\n", i+1)
		tx, err = db.Begin()
		if err != nil {
			return
		}

		err = migrations[i-1](tx)
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
	log.Info("initializing database")

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

// Create inaccessible account used for automatic internal purposes
func createSystemAccount() (err error) {
	password, err := auth.RandomID(32)
	if err != nil {
		return
	}
	hash, err := auth.BcryptHash(password, 10)
	if err != nil {
		return
	}
	return RegisterAccount("system", hash)
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

// Patches server configuration during upgrades
func patchConfigs(tx *sql.Tx, fn func(*config.Configs)) (err error) {
	var s string
	err = tx.QueryRow("SELECT val FROM main WHERE id = 'config'").Scan(&s)
	if err != nil {
		return
	}
	conf, err := decodeConfigs(s)
	if err != nil {
		return
	}

	fn(&conf)

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
}
