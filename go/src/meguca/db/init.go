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

	_ "github.com/lib/pq" // Postgres driver
)

const (
	version = 3
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
	} else { // Perform any upgrades
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
	}
	tasks = append(tasks, genPrepared)
	if !exists {
		tasks = append(tasks, CreateAdminAccount)
	}
	tasks = append(tasks, loadConfigs, loadBoardConfigs, loadBans)
	if err := util.Waterfall(tasks...); err != nil {
		return err
	}

	if !IsTest {
		go runCleanupTasks()
	}
	return nil
}

func rollBack(tx *sql.Tx, err error) error {
	if rbErr := tx.Rollback(); rbErr != nil {
		err = util.WrapError(err.Error(), rbErr)
	}
	return err
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
		if _, err := db.Exec(`DELETE FROM ` + t); err != nil {
			return err
		}
	}
	return nil
}
