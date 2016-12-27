// Initializes and loads RethinkDB

package db

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/bakape/meguca/auth"
	// Postgres driver
	"strconv"

	_ "github.com/lib/pq"
)

const dbVersion = 1

var (
	// DBName is the name of the database to use
	DBName = "meguca"

	// IsTest can be overridden to not launch several infinite loops during tests
	// or check DB version
	IsTest bool

	// AllTables are all tables needed for meguca operation
	AllTables = []string{}

	// DB stores the postgres database instance
	DB *sql.DB

	// DBPassword stores the Postgres user password
	DBPassword = "meguca"
)

// LoadDB establishes connections to RethinkDB and Redis and bootstraps both
// databases, if not yet done.
func LoadDB() (err error) {
	args := fmt.Sprintf(
		`user='meguca' password='%s' dbname='%s' sslmode=disable`,
		PDBPassword, DBName,
	)
	DB, err = sql.Open("postgres", args)
	if err != nil {
		return err
	}

	var exists bool
	err = PDB.QueryRow(`
SELECT EXISTS (
	SELECT 1
	FROM  information_schema.tables
	WHERE table_schema = 'public'
		AND table_name = 'main'
);`).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return initPostgres()
	}

	return nil

	// if !IsTest {
	// 	go runCleanupTasks()
	// }
	// return util.Waterfall([]func() error{
	// 	connectToPostgres, loadConfigs, loadConfigs,
	// })
}

// InitDB initializes a database
func InitDB() error {
	log.Printf("initializing database '%s'", DBName)
	_, err := PDB.Exec(`
BEGIN;
	CREATE TABLE 'main' (
	'id' TEXT NOT NULL,
	'val' TEXT NOT NULL,
	PRIMARY KEY ('id')
	)
	INSERT INTO 'main' ('id', 'val')
	VALUES ('version', $1);

	CREATE TABLE 'accounts' (
	'id' VARCHAR(20) NOT NULL,
	'password' BYTEA(60) NOT NULL,
	'sessions' TEXT NOT NULL,
	PRIMARY KEY ('id')
	)

	CREATE TABLE 'images' (
	'SHA1' CHAR(40) NOT NULL,
	'APNG' BOOLEAN,
	'audio' BOOLEAN,
	'video' BOOLEAN,
	'fileType' BIT(8) NOT NULL,
	'thumbType' BIT(8) NOT NULL,
	'length' INT,
	'dims' SMALLINT[4] NOT NULL,
	'size' INT NOT NULL,
	'MD5' CHAR(22) NOT NULL,
	PRIMARY KEY ('SHA1')
	)

	CREATE TABLE 'image_tokens' (
	'token' CHAR(32) NOT NULL,
	'SHA1' CHAR(40) NOT NULL,
	'expires' BIGINT NOT NULL,
	PRIMARY KEY ('token'),
	INDEX 'expires' ('expires'),
	CONSTRAINT 'image'
		FOREIGN KEY ('SHA1')
		REFERENCES 'images' ('SHA1')
		ON DELETE CASCADE
	)

	CREATE TABLE 'boards' (
	'id' VARCHAR(3) NOT NULL,
	'readOnly' BOOLEAN NOT NULL,
	'textOnly' BOOLEAN NOT NULL,
	'forcedAnon' BOOLEAN NOT NULL,
	'hashCommands' BOOLEAN NOT NULL,
	'codeTags' BOOLEAN NOT NULL,
	'title' VARCHAR(100) NOT NULL,
	'notice' VARCHAR(500) NOT NULL,
	'rules' VARCHAR(5000) NOT NULL,
	PRIMARY KEY ('id')
	)

	CREATE TABLE 'staff' (
	'board' VARCHAR(3) NOT NULL,
	'position' VARCHAR(50) NOT NULL,
	'account' VARCHAR(20) NOT NULL,
	PRIMARY KEY ('board', 'position'),
	INDEX 'account' ('account'),
	CONSTRAINT 'board'
		FOREIGN KEY ('board')
		REFERENCES 'boards' ('id')
		ON DELETE CASCADE,
	CONSTRAINT 'account'
		FOREIGN KEY ('account')
		REFERENCES 'accounts' ('id')
		ON DELETE CASCADE
	)

	CREATE TABLE 'threads' (
	'id' BIGINT NOT NULL,
	'board' VARCHAR(3) NOT NULL,
	'subject' VARCHAR(100) NOT NULL,
	'log' BYTEA[] NOT NULL,
	PRIMARY KEY ('id'),
	INDEX 'board' ('board'),
	CONSTRAINT 'board'
		FOREIGN KEY ('board')
		REFERENCES 'boards' ('id')
		ON DELETE CASCADE
	)

	CREATE TABLE 'posts' (
	'id' BIGSERIAL NOT NULL,
	'op' BIGINT NOT NULL,
	'editing' BOOLEAN NOT NULL,
	'deleted' BOOLEAN,
	'spoiler' BOOLEAN,
	'board' VARCHAR(3) NOT NULL,
	'ip' INET NOT NULL,
	'time' BIGINT NOT NULL,
	'body' VARCHAR(2000) NOT NULL,
	'postPassword' BYTEA(60) NOT NULL,
	'name' VARCHAR(50),
	'trip' CHAR(10),
	'auth' VARCHAR(20),
	'SHA1' CHAR(40),
	'imageName' VARCHAR(200),
	'commands' TEXT[],
	PRIMARY KEY ('id'),
	INDEX 'op' ('op'),
	INDEX 'editing' ('editing'),
	INDEX 'board' ('board'),
	INDEX 'ip' ('ip'),
	INDEX 'SHA1' ('SHA1'),
	CONSTRAINT 'op'
		FOREIGN KEY ('op')
		REFERENCES 'threads' ('id')
		ON DELETE CASCADE,
	CONSTRAINT 'image'
		FOREIGN KEY ('SHA1')
		REFERENCES 'images' ('SHA1')
		ON DELETE SET NULL
	)

	CREATE TABLE 'backlinks' (
	'source' BIGINT NOT NULL,
	'target' BIGINT NOT NULL,
	'targetBoard' VARCHAR(3) NOT NULL,
	PRIMARY KEY ('source'),
	INDEX 'target' ('target'),
	CONSTRAINT 'source'
		FOREIGN KEY ('source')
		REFERENCES 'posts' ('id')
		ON DELETE CASCADE,
	CONSTRAINT 'target'
		FOREIGN KEY ('target')
		REFERENCES 'posts' ('id')
		ON DELETE CASCADE
	)

	CREATE TABLE 'meguca'.'links' (
	'source' BIGINT NOT NULL,
	'target' BIGINT NOT NULL,
	'targetBoard' VARCHAR(3) NOT NULL,
	PRIMARY KEY ('source'),
	INDEX 'target' ('target'),
	CONSTRAINT 'source'
		FOREIGN KEY ('source')
		REFERENCES 'posts' ('id')
		ON DELETE CASCADE,
	CONSTRAINT 'target'
		FOREIGN KEY ('target')
		REFERENCES 'posts' ('id')
		ON DELETE CASCADE
	)
COMMIT;`,
		strconv.Itoa(dbVersion),
	)
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
	q := r.Expr(tables).ForEach(func(table r.Term) r.Term {
		return r.Table(table).Delete()
	})
	return Write(q)
}
