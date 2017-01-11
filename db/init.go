// Initializes and loads RethinkDB

package db

import (
	"database/sql"
	"fmt"
	"log"

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

	// Stores the postgres database instance
	db *sql.DB

	// DBPassword stores the Postgres user password
	DBPassword = "meguca"
)

// Initial table creation queries
const initQ = `
CREATE TABLE main (
	id TEXT PRIMARY KEY,
	val TEXT NOT NULL
);
INSERT INTO main (id, val) VALUES
	('version', %d),
	('config', '%s');

CREATE TABLE accounts (
	id VARCHAR(20) PRIMARY KEY,
	password BYTEA NOT NULL
);

CREATE TABLE sessions (
	expires BIGINT NOT NULL,
	account VARCHAR(20) NOT NULL REFERENCES accounts ON DELETE CASCADE,
	token TEXT NOT NULL,
	PRIMARY KEY (account, token)
);

CREATE TABLE images (
	APNG BOOLEAN NOT NULL,
	audio BOOLEAN NOT NULL,
	video BOOLEAN NOT NULL,
	fileType SMALLINT NOT NULL,
	thumbType SMALLINT NOT NULL,
	dims SMALLINT[4] NOT NULL,
	length INT NOT NULL,
	size INT NOT NULL,
	MD5 CHAR(22) NOT NULL,
	SHA1 CHAR(40) PRIMARY KEY
);

CREATE TABLE image_tokens (
	token CHAR(32) NOT NULL,
	SHA1 CHAR(40) NOT NULL REFERENCES images ON DELETE CASCADE,
	expires BIGINT NOT NULL
);

CREATE TABLE boards (
	readOnly BOOLEAN NOT NULL,
	textOnly BOOLEAN NOT NULL,
	forcedAnon BOOLEAN NOT NULL,
	hashCommands BOOLEAN NOT NULL,
	codeTags BOOLEAN NOT NULL,
	id VARCHAR(3) PRIMARY KEY,
	created BIGINT NOT NULL,
	ctr BIGINT DEFAULT 0,
	title VARCHAR(100) NOT NULL,
	notice VARCHAR(500) NOT NULL,
	rules VARCHAR(5000) NOT NULL,
	eightball TEXT[] NOT NULL
);

CREATE TABLE staff (
	board VARCHAR(3) NOT NULL REFERENCES boards ON DELETE CASCADE,
	account VARCHAR(20) NOT NULL REFERENCES accounts ON DELETE CASCADE,
	position VARCHAR(50) NOT NULL,
	PRIMARY KEY (board, position)
);

CREATE SEQUENCE post_id;

CREATE TABLE threads (
	board VARCHAR(3) NOT NULL REFERENCES boards ON DELETE CASCADE,
	id BIGINT PRIMARY KEY,
	postCtr BIGINT NOT NULL,
	imageCtr BIGINT NOT NULL,
	bumpTime BIGINT NOT NULL,
	replyTime BIGINT NOT NULL,
	subject VARCHAR(100) NOT NULL,
	log BYTEA[] NOT NULL
);
CREATE INDEX threads_board on threads (board);
CREATE INDEX bumpTime on threads (bumpTime);

CREATE TABLE posts (
	editing BOOLEAN NOT NULL,
	spoiler BOOLEAN,
	deleted BOOLEAN,
	banned BOOLEAN,
	id BIGINT PRIMARY KEY,
	op BIGINT NOT NULL REFERENCES threads ON DELETE CASCADE,
	time BIGINT NOT NULL,
	board VARCHAR(3) NOT NULL,
	trip CHAR(10),
	auth VARCHAR(20),
	SHA1 CHAR(40) REFERENCES images ON DELETE SET NULL,
	name VARCHAR(50),
	imageName VARCHAR(200),
	body VARCHAR(2000) NOT NULL,
	postPassword BYTEA,
	links BIGINT[][2],
	backlinks BIGINT[][2],
	commands JSON[]
);
CREATE INDEX deleted on posts (deleted);
CREATE INDEX op on posts (op);
CREATE INDEX image on posts (SHA1);
CREATE INDEX editing on posts (editing);
`

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
	db, err = sql.Open("postgres", connArgs())
	if err != nil {
		return err
	}

	var exists bool
	err = db.QueryRow(`
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
		prepared[k], err = db.Prepare(q)
		if err != nil {
			return err
		}
	}

	// if !IsTest {
	// 	go runCleanupTasks()
	// }

	return util.Waterfall(loadConfigs, loadBoardConfigs)
}

// InitDB initializes a database
func InitDB() error {
	log.Printf("initializing database '%s'", DBName)

	conf, err := ffjson.Marshal(config.Defaults)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(fmt.Sprintf(initQ, version, string(conf)))
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
		if _, err := db.Exec(`DELETE FROM ` + t); err != nil {
			return err
		}
	}
	return nil
}
