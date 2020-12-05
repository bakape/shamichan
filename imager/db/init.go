package db

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/bakape/meguca/imager/common"
	"github.com/bakape/meguca/imager/config"
	"github.com/jackc/pgx/v4/pgxpool"
)

var (
	// ConnArgs specifies the PostgreSQL connection URL
	connectionURL string

	// Postgres connection pool
	db *pgxpool.Pool
)

// Connects to PostgreSQL database and performs schema upgrades
func LoadDB() error {
	return loadDB(config.Server.Database, "")
}

// Create and load testing database
func LoadTestDB() (err error) {
	_, path, _, ok := runtime.Caller(1)
	if !ok {
		panic("could not get caller file")
	}
	dir, _ := filepath.Split(path)
	suffix := filepath.Base(dir)

	common.IsTest = true

	srcURL := os.Getenv("TEST_DB")
	if srcURL == "" {
		srcURL = config.Server.Database
	}
	connURL, err := url.Parse(srcURL)
	if err != nil {
		return
	}

	run := func(line ...string) error {
		c := exec.Command(line[0], line[1:]...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	}
	user := connURL.User.Username()
	dbName := fmt.Sprintf("%s_test_%s", strings.Trim(connURL.Path, "/"), suffix)

	err = run(
		"psql",
		"-c", "drop database if exists "+dbName,
		srcURL,
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
		srcURL,
	)
	if err != nil {
		return
	}

	connURL.Path = "/" + dbName
	err = loadDB(connURL.String(), suffix)
	return
}

func loadDB(connURL, dbSuffix string) (err error) {
	// Set, for creating extra connections using Listen()
	connectionURL = connURL

	u, err := url.Parse(connURL)
	if err != nil {
		return
	}
	q := u.Query()
	q.Set("pool_max_conns", "50")
	q.Set("sslmode", "disable")
	u.RawQuery = q.Encode()

	db, err = pgxpool.Connect(context.Background(), u.String())
	if err != nil {
		return
	}

	err = loadConfig(context.Background())
	if err != nil {
		return
	}

	return
}

// Close DB and release resources
func Close() (err error) {
	db.Close()
	return nil
}

// ClearTables deletes the contents of specified DB tables. Only used for tests.
func ClearTables(tables ...string) (err error) {
	for _, t := range tables {
		_, err = db.Exec(context.Background(), `delete from `+t)
		if err != nil {
			return
		}
	}
	return
}
