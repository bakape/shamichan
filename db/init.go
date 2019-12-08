package db

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
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
func LoadTestDB() (err error) {
	_, path, _, ok := runtime.Caller(1)
	if !ok {
		panic("could not get caller file")
	}
	dir, _ := filepath.Split(path)
	suffix := filepath.Base(dir)

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
	// Set, for creating extra connections using Listen()
	connectionURL = connURL

	connOpts, err := pgx.ParseURI(connURL)
	if err != nil {
		return
	}
	db, err = pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig:     connOpts,
		MaxConnections: 50,
	})
	if err != nil {
		return
	}

	err = runMigrations()
	if err != nil {
		return
	}
	err = loadConfigs()
	if err != nil {
		return
	}

	// TODO: Reenable this
	// err = loadThreadPostCounts()
	// if err != nil {
	// 	return
	// }
	if !common.IsTest {
		go runCleanupTasks()
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
	clearOpenPostBuffer() // Clear Open post buffer between tests
	for _, t := range tables {
		_, err = db.Exec(`DELETE FROM ` + t)
		if err != nil {
			return
		}
	}
	return
}
