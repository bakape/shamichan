// Initializes and loads RethinkDB

package db

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/util"
	_ "github.com/lib/pq" // Postgres driver
	"github.com/pquerna/ffjson/ffjson"
)

const version = 1

var (
	// DBName is the name of the database to use
	DBName = "meguca"

	// IsTest can be overridden to not launch several infinite loops during tests
	// or check DB version
	IsTest bool

	// DB stores the postgres database instance
	DB *sql.DB

	// DBPassword stores the Postgres user password
	DBPassword = "meguca"

	// Queries to be converted into prepared statements
	protoPrepared = map[string]string{}

	// Prepared stores generated prepared statements
	Prepared = map[string]*sql.Stmt{}
)

// Initial table creation queries
var initQ = [...]string{
	`
CREATE TABLE main (
	id TEXT
		PRIMARY KEY,
	val TEXT
		NOT NULL
);`, `
INSERT INTO main (id, val) VALUES
	('version', $1),
	('config', $2);`, `
CREATE TABLE accounts (
	id VARCHAR(20)
		PRIMARY KEY,
	password BYTEA
		NOT NULL
);`, `
CREATE TABLE sessions (
	account VARCHAR(20)
		REFERENCES accounts
		ON DELETE CASCADE,
	token TEXT,
	expires BIGINT
		NOT NULL,
	PRIMARY KEY (account, token)
);`, `
CREATE TABLE images (
	APNG BOOLEAN,
	audio BOOLEAN,
	video BOOLEAN,
	fileType BIT(8)
		NOT NULL,
	thumbType BIT(8)
		NOT NULL,
	dims SMALLINT[4]
		NOT NULL,
	length INT,
	size INT
		NOT NULL,
	SHA1 CHAR(40)
		PRIMARY KEY,
	MD5 CHAR(22)
		NOT NULL
);`, `
CREATE TABLE image_tokens (
	token CHAR(32)
		NOT NULL,
	SHA1 CHAR(40)
		NOT NULL
		REFERENCES images
		ON DELETE CASCADE,
	expires BIGINT
		NOT NULL
);`, `
CREATE TABLE boards (
	readOnly BOOLEAN
		NOT NULL,
	textOnly BOOLEAN
		NOT NULL,
	forcedAnon BOOLEAN
		NOT NULL,
	hashCommands BOOLEAN
		NOT NULL,
	id VARCHAR(3)
		PRIMARY KEY,
	codeTags BOOLEAN
		NOT NULL,
	title VARCHAR(100)
		NOT NULL,
	notice VARCHAR(500)
		NOT NULL,
	rules VARCHAR(5000)
		NOT NULL,
	eightball TEXT[]
		NOT NULL
);`, `
CREATE TABLE staff (
	board VARCHAR(3)
		NOT NULL
		REFERENCES boards
		ON DELETE CASCADE,
	account VARCHAR(20)
		NOT NULL
		REFERENCES accounts
		ON DELETE CASCADE,
	position VARCHAR(50)
		NOT NULL,
	PRIMARY KEY (board, position)
);`, `
CREATE TABLE threads (
	board VARCHAR(3)
		NOT NULL
		REFERENCES boards
		ON DELETE CASCADE,
	log BYTEA[]
		NOT NULL,
	id BIGINT
		PRIMARY KEY,
	subject VARCHAR(100)
		NOT NULL
);`, `
CREATE INDEX threads_board on threads (board);`, `
CREATE TABLE posts (
	editing BOOLEAN
		NOT NULL,
	deleted BOOLEAN,
	spoiler BOOLEAN,
	board VARCHAR(3)
		NOT NULL,
	ip INET
		NOT NULL,
	id BIGSERIAL
		PRIMARY KEY,
	op BIGINT
		NOT NULL
		REFERENCES threads
		ON DELETE CASCADE,
	time BIGINT
		NOT NULL,
	body VARCHAR(2000)
		NOT NULL,
	postPassword BYTEA
		NOT NULL,
	name VARCHAR(50),
	trip CHAR(10),
	auth VARCHAR(20),
	SHA1 CHAR(40)
		REFERENCES images
		ON DELETE SET NULL,
	imageName VARCHAR(200),
	commands TEXT[]
);`, `
CREATE INDEX op on posts (op);`, `
CREATE INDEX image on posts (SHA1);`, `
CREATE INDEX editing on posts (editing);`, `
CREATE INDEX ip on posts (ip);`, `
CREATE TABLE links (
	targetBoard VARCHAR(3)
		NOT NULL,
	source BIGINT
		PRIMARY KEY
		REFERENCES posts
		ON DELETE CASCADE,
	target BIGINT
		NOT NULL
		REFERENCES posts
		ON DELETE CASCADE
);`, `
CREATE INDEX links_source on links (source);`, `
CREATE INDEX links_target on links (target);`, `
CREATE TABLE backlinks (
	targetBoard VARCHAR(3)
		NOT NULL,
	source BIGINT
		PRIMARY KEY
		REFERENCES posts
		ON DELETE CASCADE,
	target BIGINT
		NOT NULL
		REFERENCES posts
		ON DELETE CASCADE
);`, `
CREATE INDEX backlinks_source on backlinks (source);`, `
CREATE INDEX backlinks_target on backlinks (target);`,
}

// Generates a Postgres connection parameter string
func connArgs() string {
	return fmt.Sprintf(
		`user='meguca' password='%s' dbname='%s' sslmode=disable`,
		DBPassword, DBName,
	)
}

// LoadDB establishes connections to RethinkDB and Redis and bootstraps both
// databases, if not yet done.
func LoadDB() (err error) {
	DB, err = sql.Open("postgres", connArgs())
	if err != nil {
		return err
	}

	var exists bool
	err = DB.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM  information_schema.tables
			WHERE table_schema = 'public'
				AND table_name = 'main'
		);`).
		Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		if err := InitDB(); err != nil {
			return err
		}
	}

	// Generate prepared statements
	for k, q := range protoPrepared {
		Prepared[k], err = DB.Prepare(q)
		if err != nil {
			return err
		}
	}

	// if !IsTest {
	// 	go runCleanupTasks()
	// }

	return util.Waterfall([]func() error{
		loadConfigs, loadBoardConfigs,
	})
}

// InitDB initializes a database
func InitDB() error {
	log.Printf("initializing database '%s'", DBName)

	tx, err := DB.Begin()
	if err != nil {
		return err
	}

	exec := func(q string, args ...interface{}) {
		if err != nil {
			return
		}
		_, err = tx.Exec(q, args...)
	}

	// Init main table
	exec(initQ[0])
	conf, err := ffjson.Marshal(config.Defaults)
	if err != nil {
		tx.Rollback()
		return err
	}
	exec(initQ[1], strconv.Itoa(version), string(conf))

	for i := 2; i < len(initQ); i++ {
		exec(initQ[i])
	}

	if err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return CreateAdminAccount()
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
		if _, err := DB.Exec(`DELETE FROM ` + t); err != nil {
			return err
		}
	}
	return nil
}

// AddPrepared adds a prepared statement to be generated after connecting to the
// DB.
// Not thread safe. Should only be called from init() functions.
func AddPrepared(key, query string) {
	protoPrepared[key] = query
}
