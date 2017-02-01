// Initializes and loads RethinkDB

package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/util"
	_ "github.com/lib/pq" // Postgres driver
)

const (
	version = 1
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
