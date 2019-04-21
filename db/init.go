package db

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/util"
	"github.com/boltdb/bolt"
	"github.com/go-playground/log"
	_ "github.com/lib/pq" // Postgres driver
)

const (
	// DefaultConnArgs specifies the default PostgreSQL connection arguments
	DefaultConnArgs = "user=meguca password=meguca dbname=meguca sslmode=disable binary_parameters=yes"
)

var (
	// ConnArgs specifies the PostgreSQL connection arguments
	ConnArgs string

	// Stores the postgres database instance
	db *sql.DB

	// Statement builder and cacher
	sq squirrel.StatementBuilderType

	// Embedded database for temporary storage
	boltDB *bolt.DB
)

// Connects to PostgreSQL database and performs schema upgrades
func LoadDB() error {
	return loadDB("")
}

// Create and load testing database. Call close() to clean up temporary
// resources.
func LoadTestDB(suffix string) (close func() error, err error) {
	common.IsTest = true

	// If running as root (CI like Travis or something), authenticate as the
	// postgres user
	var u *user.User
	u, err = user.Current()
	if err != nil {
		return
	}
	var sudo []string
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

	suffix = "_" + suffix
	name := "meguca_test" + suffix

	err = run("psql",
		"-U", user,
		"-c", "drop database if exists "+name)
	if err != nil {
		return
	}

	close = func() (err error) {
		err = boltDB.Close()
		if err != nil {
			return
		}
		return os.Remove(fmt.Sprintf("db%s.db", suffix))
	}

	fmt.Println("creating test database:", name)
	err = run("createdb",
		"-O", "meguca",
		"-U", user,
		"-E", "UTF8",
		name)
	if err != nil {
		return
	}

	ConnArgs = fmt.Sprintf(
		"postgres://%s@localhost:5432/%s?sslmode=disable&binary_parameters=yes",
		user, name)
	err = loadDB(suffix)
	return
}

func loadDB(dbSuffix string) (err error) {
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
	} else if err = runMigrations(); err != nil {
		return
	}

	// Run these is parallel
	tasks = append(
		tasks,
		func() error {
			tasks := []func() error{loadConfigs, loadBans, handleSpamScores}
			if config.ImagerMode != config.ImagerOnly {
				tasks = append(tasks, openBoltDB(dbSuffix), loadBanners,
					loadLoadingAnimations, loadThreadPostCounts)
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

	if !common.IsTest {
		go runCleanupTasks()
	}

	return nil
}

func openBoltDB(dbSuffix string) func() error {
	return func() (err error) {
		boltDB, err = bolt.Open(
			fmt.Sprintf("db%s.db", dbSuffix),
			0600,
			&bolt.Options{
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
}

// initDB initializes a database
func initDB() (err error) {
	log.Info("initializing database")

	_, err = db.Exec(
		`create table main (
			id text primary key,
			val text not null
		)`,
	)
	if err != nil {
		return
	}
	_, err = db.Exec(
		`insert into main (id, val)
		values ('version', '0'),
				('pyu', '0')`,
	)
	if err != nil {
		return
	}
	return runMigrations()
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
