package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"meguca/auth"
	"meguca/common"
	"meguca/config"
	"meguca/util"
	"time"

	"github.com/Masterminds/squirrel"

	"github.com/boltdb/bolt"
	_ "github.com/lib/pq" // Postgres driver
)

const (
	// TestConnArgs contains ConnArgs used for tests
	TestConnArgs = `user=meguca password=meguca dbname=meguca_test sslmode=disable binary_parameters=yes`
)

var (
	version = len(upgrades) + 1

	// ConnArgs specifies the PostgreSQL connection arguments
	ConnArgs = `user=meguca password=meguca dbname=meguca sslmode=disable binary_parameters=yes`

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

var upgrades = []func(*sql.Tx) error{
	func(tx *sql.Tx) (err error) {
		// Delete legacy columns
		return execAll(tx,
			`ALTER TABLE threads
				DROP COLUMN locked`,
			`ALTER TABLE boards
				DROP COLUMN hashCommands,
				DROP COLUMN codeTags`,
		)
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE threads
				DROP COLUMN log`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE boards
				DROP COLUMN ctr`,
		)
		return
	},
	// Restore correct image counters after incorrect updates
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`UPDATE threads
				SET imageCtr = (SELECT COUNT(*) FROM posts
					WHERE SHA1 IS NOT NULL
						AND op = threads.id
				)`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE images
				ADD COLUMN Title varchar(100) not null default '',
				ADD COLUMN Artist varchar(100) not null default ''`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE posts
				ADD COLUMN sage bool`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(`DROP INDEX deleted`)
		return
	},
	// Set default expiry configs, to keep all threads from deleting
	func(tx *sql.Tx) (err error) {
		return patchConfigs(tx, func(conf *config.Configs) {
			conf.ThreadExpiryMin = config.Defaults.ThreadExpiryMin
			conf.ThreadExpiryMax = config.Defaults.ThreadExpiryMax
		})
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE boards
				ADD COLUMN disableRobots bool default false`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE threads
				ADD COLUMN sticky bool default false`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE bans
				ADD COLUMN forPost bigint default 0`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		return execAll(tx,
			`create table mod_log (
				type smallint not null,
				board varchar(3) not null,
				id bigint not null,
				by varchar(20) not null,
				created timestamp default (now() at time zone 'utc')
			)`,
			`create index mod_log_board on mod_log (board)`,
			`create index mod_log_created on mod_log (created)`,
		)
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(`create index sticky on threads (sticky)`)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE posts
				DROP COLUMN backlinks`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`create table banners (
				board varchar(3) not null references boards on delete cascade,
				id smallint not null,
				data bytea not null,
				mime text not null
			);`,
		)
		return
	},
	func(tx *sql.Tx) error {
		return execAll(tx,
			`alter table boards
				alter column id type text`,
			`alter table bans
				alter column board type text`,
			`alter table mod_log
				alter column board type text`,
			`alter table staff
				alter column board type text`,
			`alter table banners
				alter column board type text`,
			`alter table threads
				alter column board type text`,
			`alter table posts
				alter column board type text`,
		)
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`create table loading_animations (
				board text primary key references boards on delete cascade,
				data bytea not null,
				mime text not null
			);`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE boards
				ADD COLUMN defaultCSS text default 'moe'`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE posts
				ADD COLUMN flag char(2)`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE boards
				ADD COLUMN flags bool default false`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`alter table images
				alter column title type varchar(200)`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE boards
				ADD COLUMN NSFW bool default false`,
		)
		return
	},
	func(tx *sql.Tx) error {
		return execAll(tx,
			`create table reports (
				id bigserial primary key,
				target bigint not null,
				board text not null,
				reason text not null,
				by inet not null,
				illegal boolean not null,
				created timestamp default (now() at time zone 'utc')
			)`,
			`create index report_board on reports (board)`,
			`create index report_created on reports (created)`,
		)
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE boards
				ADD COLUMN nonLive bool default false`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE threads
				ADD COLUMN nonLive bool default false`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE boards
				ADD COLUMN posterIDs bool default false`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE posts
				ADD COLUMN posterID text`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE threads
				ADD COLUMN locked bool default false`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE boards
				ADD COLUMN js varchar(5000) default ''`,
		)
		return
	},
	// Fix consequences of bug in init.sql
	func(tx *sql.Tx) (err error) {
		q := `SELECT EXISTS (SELECT 1
			FROM information_schema.columns
			WHERE table_schema='public'
				AND table_name='posts'
				AND column_name='locked'
		)`
		var exists bool
		err = tx.QueryRow(q).Scan(&exists)
		if err != nil || !exists {
			return
		}

		// Correct it
		_, err = tx.Exec(
			`ALTER TABLE posts
				DROP COLUMN locked`,
		)
		if err != nil {
			return
		}
		_, err = tx.Exec(
			`ALTER TABLE threads
				ADD COLUMN locked bool default false`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE boards
				DROP COLUMN js`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		return execAll(tx,
			`DROP INDEX editing`,
			`ALTER TABLE boards
				DROP COLUMN nonLive`,
			`ALTER TABLE posts
				DROP COLUMN editing`,
		)
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE posts
				DROP COLUMN password`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		return execAll(tx,
			`ALTER TABLE boards
				ADD COLUMN nonLive bool default false`,
			`ALTER TABLE posts
				ADD COLUMN 	editing boolean not null default false`,
			`create index editing on posts (editing);`,
		)
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE posts
				ADD COLUMN password bytea`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`INSERT INTO main VALUES ('roulette', '6')`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		// Fuck any wise guy trying to create an account nad block an upgrade
		_, err = tx.Exec(
			`DELETE FROM accounts
			WHERE id = 'system'`,
		)
		if err != nil {
			return
		}

		password, err := auth.RandomID(32)
		if err != nil {
			return
		}
		hash, err := auth.BcryptHash(password, 10)
		if err != nil {
			return
		}
		_, err = tx.Exec(
			`insert into accounts (id, password)
			values ('system', $1)`,
			hash,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		return patchConfigs(tx, func(conf *config.Configs) {
			conf.CharScore = config.Defaults.CharScore
			conf.PostCreationScore = config.Defaults.PostCreationScore
			conf.ImageScore = config.Defaults.ImageScore
		})
	},
	func(tx *sql.Tx) (err error) {
		var rcount string
		err = tx.QueryRow(
			`SELECT COUNT(*) FROM mod_log WHERE board != 'all' AND by = 'system' AND type = 0`,
		).Scan(&rcount)
		if err != nil {
			return
		}
		_, err = tx.Exec(
			`INSERT INTO main VALUES ('rcount', $1)`,
			rcount,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		execAll(tx,
			`create table links (
				source bigint not null references posts on delete cascade,
				target bigint not null references posts on delete cascade,
				primary key(source, target)
			);`,
			`create index link_source on links (source);`,
			`create index link_target on links (target);`,
		)
		if err != nil {
			return
		}

		// Read all posts and links
		r, err := tx.Query(
			`select p.id, p.op, p.links from posts as p
			join threads as t on t.id = p.op`,
		)
		if err != nil {
			return
		}
		defer r.Close()
		var (
			posts  = make(map[uint64]bool, 1<<10)
			links  = make(map[uint64]uint64, 1<<10)
			id, op uint64
			lr     linkRow
		)
		for r.Next() {
			err = r.Scan(&id, &op, &lr)
			if err != nil {
				return
			}
			posts[id] = true
			for _, pair := range lr {
				links[id] = pair[0]
			}
		}
		if err != nil {
			return
		}

		// Remove legacy link row
		_, err = tx.Exec(`alter table posts drop column links`)
		if err != nil {
			return
		}

		// Write only verified links to new table
		q, err := tx.Prepare(
			`insert into links (source, target)
			values ($1, $2)`,
		)
		if err != nil {
			return
		}
		for source, target := range links {
			if !posts[source] || !posts[target] {
				continue
			}
			_, err = q.Exec(source, target)
			if err != nil {
				return
			}
		}

		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(`ALTER TABLE banners DROP COLUMN id`)
		return
	},
	func(tx *sql.Tx) error {
		return execAll(tx,
			`DELETE FROM mod_log`,
			`DELETE FROM bans`,
			`ALTER TABLE mod_log ADD CONSTRAINT mod_log_board_fkey
			FOREIGN KEY (board) REFERENCES boards(id) ON DELETE CASCADE`,
			`ALTER TABLE bans ADD CONSTRAINT bans_board_fkey
			FOREIGN KEY (board) REFERENCES boards(id) ON DELETE CASCADE`,
		)
	},
	func(tx *sql.Tx) error {
		return execAll(tx,
			`ALTER TABLE threads DROP COLUMN postCtr`,
			`ALTER TABLE threads DROP COLUMN imageCtr`,
		)
	},
	func(tx *sql.Tx) error {
		return execAll(tx,
			`ALTER TABLE mod_log ADD COLUMN length bigint default 0`,
			`ALTER TABLE mod_log ADD COLUMN reason text default ''`,
		)
	},
	func(tx *sql.Tx) error {
		return execAll(tx,
			`create index image_fileType on images (fileType)`,
			`create index image_audio on images (audio)`,
			`create index post_board on posts (board)`,
		)
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE boards
				ADD COLUMN rbText bool default false`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		// Read all commands
		r, err := withTransaction(tx,
			sq.Select("id", "commands").
				From("posts").
				Where("commands is not null"),
		).
			Query()
		if err != nil {
			return
		}
		comms := make(map[uint64][]common.Command, 1024)
		var id uint64
		for r.Next() {
			var com commandRow
			err = r.Scan(&id, &com)
			if err != nil {
				return
			}
			comms[id] = []common.Command(com)
		}
		err = r.Err()
		if err != nil {
			return
		}

		prep, err := tx.Prepare(
			`update posts
			set commands = $2
			where id = $1`)
		if err != nil {
			return
		}

		// Remove all #pyu/#pcount commands
		new := make(commandRow, 0, 64)
		for id, comms := range comms {
			new = new[:0]
			for _, c := range comms {
				switch c.Type {
				case common.Pyu, common.Pcount:
				default:
					new = append(new, c)
				}
			}
			val := new
			if len(new) == 0 {
				val = nil
			}
			_, err = prep.Exec(id, val)
			if err != nil {
				return
			}
		}

		return
	},
	func(tx *sql.Tx) error {
		return withTransaction(tx, sq.Delete("main").Where("id = 'pyu'")).Exec()
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`ALTER TABLE boards
				ADD COLUMN pyu bool default false`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(
			`create table pyu (
				id text primary key references boards on delete cascade,
				pcount bigint default 0
			);
			create table pyu_limit (
				ip inet not null,
				board text not null references boards on delete cascade,
				expires timestamp not null,
				pcount smallint default 4,
				primary key(ip, board)
			);
			create index pyu_limit_ip on pyu_limit (ip);
			create index pyu_limit_board on pyu_limit (board);`,
		)
		return
	},
	func(tx *sql.Tx) (err error) {
		// Reverted migration
		return
	},
	func(tx *sql.Tx) (err error) {
		// Revert changes
		return execAll(tx,
			`alter table boards drop column pyu`,
			`drop table pyu`,
			`drop table pyu_limit`
		)
	},
}

// LoadDB establishes connections to RethinkDB and Redis and bootstraps both
// databases, if not yet done.
func LoadDB() (err error) {
	db, err = sql.Open("postgres", ConnArgs)
	if err != nil {
		return
	}

	sq = squirrel.StatementBuilder.
		RunWith(squirrel.NewStmtCacheProxy(db)).
		PlaceholderFormat(squirrel.Dollar)

	var exists bool
	err = db.QueryRow(getQuery("init/check_db_exists.sql")).Scan(&exists)
	if err != nil {
		return
	}

	tasks := make([]func() error, 0, 16)
	if !exists {
		tasks = append(tasks, initDB)
	} else if err = checkVersion(); err != nil {
		return
	}
	tasks = append(tasks, genPrepared)

	// Run these is parallel
	tasks = append(
		tasks,
		func() error {
			tasks := []func() error{
				openBoltDB, loadConfigs, loadBans,
				loadBanners, loadLoadingAnimations,
			}
			if !exists {
				tasks = append(tasks, CreateAdminAccount, createSystemAccount)
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
	if err != nil {
		return
	}

	var tx *sql.Tx
	for i := v; i < version; i++ {
		log.Printf("upgrading database to version %d\n", i+1)
		tx, err = db.Begin()
		if err != nil {
			return
		}

		err = upgrades[i-1](tx)
		if err != nil {
			return rollBack(tx, err)
		}

		// Write new version number
		_, err = tx.Exec(
			`update main set val = $1 where id = 'version'`,
			i+1,
		)
		if err != nil {
			return rollBack(tx, err)
		}

		err = tx.Commit()
		if err != nil {
			return
		}
	}

	return
}

func rollBack(tx *sql.Tx, err error) error {
	if rbErr := tx.Rollback(); rbErr != nil {
		err = util.WrapError(err.Error(), rbErr)
	}
	return err
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

// Create inaccessible account used for automatic internal purposes
func createSystemAccount() (err error) {
	password, err := auth.RandomID(32)
	if err != nil {
		return
	}
	hash, err := auth.BcryptHash(password, 10)
	if err != nil {
		return
	}
	return RegisterAccount("system", hash)
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

// Patches server configuration during upgrades
func patchConfigs(tx *sql.Tx, fn func(*config.Configs)) (err error) {
	var s string
	err = tx.QueryRow("SELECT val FROM main WHERE id = 'config'").Scan(&s)
	if err != nil {
		return
	}
	conf, err := decodeConfigs(s)
	if err != nil {
		return
	}

	fn(&conf)

	buf, err := json.Marshal(conf)
	if err != nil {
		return
	}
	_, err = tx.Exec(
		`UPDATE main
			SET val = $1
			WHERE id = 'config'`,
		string(buf),
	)
	return
}
