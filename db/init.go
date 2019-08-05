package db

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/bakape/meguca/static"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/util"
	"github.com/go-playground/log"
	"github.com/jackc/pgx"
)

var (
	// ConnArgs specifies the PostgreSQL connection URL
	connectionURL string

	// Postgres connection pool
	db *pgx.ConnPool
)

// Connects to PostgreSQL database and performs schema upgrades
func LoadDB() error {
	return loadDB(config.Server.Database, "")
}

// Create and load testing database
func LoadTestDB(suffix string) (err error) {
	common.IsTest = true

	run := func(line ...string) error {
		c := exec.Command(line[0], line[1:]...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	}
	connURL, err := url.Parse(config.Server.Test.Database)
	if err != nil {
		return
	}
	user := connURL.User.Username()
	dbName := fmt.Sprintf("%s_%s", strings.Trim(connURL.Path, "/"), suffix)

	err = run(
		"psql",
		"-c", "drop database if exists "+dbName,
		config.Server.Database,
	)
	if err != nil {
		return
	}

	fmt.Println("creating test database:", dbName)
	err = run(
		"psql",
		"-c",
		fmt.Sprintf(
			"create database %s with owner %s encoding UTF8",
			dbName, user,
		),
		config.Server.Database,
	)
	if err != nil {
		return
	}

	connURL.Path = "/" + dbName
	err = loadDB(connURL.String(), suffix)
	return
}

func loadDB(connURL, dbSuffix string) (err error) {
	// Enable binary parameters for more efficient encoding of []byte
	u, err := url.Parse(connURL)
	if err != nil {
		return
	}
	u.Query().Set("binary_parameters", "yes")
	connURL = u.String()

	// Set, for creating extra connections using Listen()
	connectionURL = connURL

	connOpts, err := pgx.ParseURI(connURL)
	if err != nil {
		return
	}
	db, err = pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig:     connOpts,
		MaxConnections: 50,
		AfterConnect: func(c *pgx.Conn) (err error) {
			return static.Walk(
				"/prepared_statements",
				func(path string, info os.FileInfo, wlkErr error) (err error) {
					if wlkErr != nil {
						return wlkErr
					}
					if info.IsDir() {
						return
					}

					buf, err := static.ReadFile(path)
					if err != nil {
						return
					}
					name := info.Name()
					i := strings.LastIndexByte(name, '.')
					_, err = c.Prepare(name[:i], string(buf))
					return
				},
			)
		},
	})

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
		func() (err error) {
			tasks := []func() error{loadConfigs, loadBans}
			if config.Server.ImagerMode != config.ImagerOnly {
				tasks = append(tasks,
					loadBanners,
					loadLoadingAnimations,
					loadThreadPostCounts,
				)
			}
			err = util.Parallel(tasks...)
			if err != nil {
				return
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

// Close DB and release resources
func Close() (err error) {
	db.Close()
	return nil
}

// CreateAdminAccount writes a fresh admin account with the default password to
// the database
func CreateAdminAccount() (err error) {
	hash, err := auth.BcryptHash("password", 10)
	if err != nil {
		return err
	}
	_, err = db.Exec(
		`insert into accounts (id, passoword)
		('admin', $1)`,
		hash,
	)
	return
}

// CreateSystemAccount create an inaccessible account used for automatic internal purposes
func CreateSystemAccount(tx *pgx.Tx) (err error) {
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
func ClearTables(tables ...string) (err error) {
	clearOpenPostBuffer() // Clear Open post buffer between tests
	for _, t := range tables {
		_, err = db.Exec(`DELETE FROM ` + t)
		if err != nil {
			return
		}
	}
	return
}
