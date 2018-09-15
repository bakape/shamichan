package db

import (
	"database/sql"
	"meguca/auth"
	"meguca/config"
	"meguca/util"
	"os"
	"os/exec"
	"os/user"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/boltdb/bolt"
	"github.com/go-playground/log"
	_ "github.com/lib/pq" // Postgres driver
)

const (
	// TestConnArgs contains ConnArgs used for tests
	TestConnArgs = `user=meguca password=meguca dbname=meguca_test sslmode=disable binary_parameters=yes`

	// DefaultConnArgs specifies the default PostgreSQL connection arguments
	DefaultConnArgs = "user=meguca password=meguca dbname=meguca sslmode=disable binary_parameters=yes"
)

var (
	// ConnArgs specifies the PostgreSQL connection arguments
	ConnArgs string

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
	if IsTest {
		log.Info("dropping previous test database")

		// If running as root (CI like Travis or something), authenticate as the
		// postgres user
		var u *user.User
		u, err = user.Current()
		if err != nil {
			return
		}
		sudo := []string{}
		user := "meguca"
		if u.Name == "root" {
			sudo = append(sudo, "sudo", "-u", "postgres")
			user = "postgres"
		}

		run := func(args ...string) error {
			line := append(sudo, args...)
			c := exec.Command(line[0], line[1:]...)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		}

		err = run("psql",
			"-U", user,
			"-c", "drop database if exists meguca_test")
		if err != nil {
			return
		}

		err = run("createdb",
			"-O", "meguca",
			"-U", user,
			"-E", "UTF8",
			"meguca_test")
		if err != nil {
			return
		}
	}

	db, err = sql.Open("postgres", ConnArgs)
	if err != nil {
		return
	}

	sq = squirrel.StatementBuilder.
		RunWith(squirrel.NewStmtCacheProxy(db)).
		PlaceholderFormat(squirrel.Dollar)

	var exists bool
	const q = `select exists (
			select 1 from information_schema.tables
				where table_schema = 'public' and table_name = 'main'
		)`
	err = db.QueryRow(q).Scan(&exists)
	if err != nil {
		return
	}

	tasks := make([]func() error, 0, 16)
	if !exists {
		tasks = append(tasks, initDB)
	} else if err = checkVersion(); err != nil {
		return
	}

	// Run these is parallel
	tasks = append(
		tasks,
		func() error {
			tasks := []func() error{loadConfigs, initCaptchas, loadBans, handleSpamScores}
			if config.ImagerMode != config.ImagerOnly {
				tasks = append(tasks, openBoltDB, loadBanners,
					loadLoadingAnimations)
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
	switch err {
	case nil, sql.ErrNoRows:
		return runMigrations(v, version)
	default:
		return
	}
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
	return runMigrations(0, version)
}

// CreateAdminAccount writes a fresh admin account with the default password to
// the database
func CreateAdminAccount(tx *sql.Tx) (err error) {
	hash, err := auth.BcryptHash("password", 10)
	if err != nil {
		return err
	}
	return RegisterAccount(tx, "admin", hash)
}

// CreateSystemAccount create an inaccessible account used for automatic internal purposes
func CreateSystemAccount(tx *sql.Tx) (err error) {
	password, err := auth.RandomID(32)
	if err != nil {
		return
	}
	hash, err := auth.BcryptHash(password, 10)
	if err != nil {
		return
	}
	return RegisterAccount(tx, "system", hash)
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
