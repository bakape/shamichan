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

const version = 1

var (
	// ConnArgs specifies the PostgreSQL connection arguments
	ConnArgs = `user=meguca password=meguca dbname=meguca sslmode=disable`

	// IsTest can be overridden to not launch several infinite loops during tests
	// or check DB version
	IsTest bool

	// Stores the postgres database instance
	db *sql.DB
)

// Initial table creation queries
const initQ = `
create table main (
	id text primary key,
	val text not null
);
insert into main (id, val) values
	('version', %d),
	('config', '%s'),
	('pyu', '0');

create table accounts (
	id varchar(20) primary key,
	password bytea not null
);

create table sessions (
	account varchar(20) not null references accounts on delete cascade,
	token text not null,
	expires timestamp not null,
	primary key (account, token)
);

create table bans (
	board varchar(3) not null,
	ip inet not null,
	by varchar(20) not null,
	reason text not null,
	expires timestamp default now(),
	primary key (ip, board)
);

create table images (
	apng boolean not null,
	audio boolean not null,
	video boolean not null,
	fileType smallint not null,
	thumbType smallint not null,
	dims smallint[4] not null,
	length int not null,
	size int not null,
	md5 char(22) not null,
	sha1 char(40) primary key
);

create table image_tokens (
	token char(86) not null primary key,
	sha1 char(40) not null references images on delete cascade,
	expires timestamp not null
);

create table boards (
	readOnly boolean not null,
	textOnly boolean not null,
	forcedAnon boolean not null,
	hashCommands boolean not null,
	codeTags boolean not null,
	id varchar(3) primary key,
	ctr bigint default 0,
	created timestamp not null,
	title varchar(100) not null,
	notice varchar(500) not null,
	rules varchar(5000) not null,
	eightball text[] not null
);

create table staff (
	board varchar(3) not null references boards on delete cascade,
	account varchar(20) not null references accounts on delete cascade,
	position varchar(50) not null
);
create index staff_board on staff (board);
create index staff_account on staff (account);

create sequence post_id;

create table threads (
	locked boolean,
	board varchar(3) not null references boards on delete cascade,
	id bigint primary key,
	postCtr bigint not null,
	imageCtr bigint not null,
	bumpTime bigint not null,
	replyTime bigint not null,
	subject varchar(100) not null,
	log text[] not null
);
create index threads_board on threads (board);
create index bumpTime on threads (bumpTime);

create table posts (
	editing boolean not null,
	spoiler boolean,
	deleted boolean,
	banned boolean,
	id bigint primary key,
	op bigint not null references threads on delete cascade,
	time bigint not null,
	board varchar(3) not null,
	trip char(10),
	auth varchar(20),
	sha1 char(40) references images on delete set null,
	name varchar(50),
	imageName varchar(200),
	body varchar(2000) not null,
	password bytea,
	ip inet,
	links bigint[][2],
	backlinks bigint[][2],
	commands json[]
);
create index deleted on posts (deleted);
create index op on posts (op);
create index image on posts (sha1);
create index editing on posts (editing);
create index ip on posts (ip);
`

// LoadDB establishes connections to RethinkDB and Redis and bootstraps both
// databases, if not yet done.
func LoadDB() (err error) {
	db, err = sql.Open("postgres", ConnArgs)
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

	if !IsTest {
		go runCleanupTasks()
	}

	return util.Waterfall(loadConfigs, loadBoardConfigs, loadBans)
}

// InitDB initializes a database
func InitDB() error {
	log.Println("initializing database")

	conf, err := json.Marshal(config.Defaults)
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
