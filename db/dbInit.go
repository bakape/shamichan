// Initializes and loads RethinkDB

package db

import (
	"fmt"
	"log"
	"time"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/util"
	r "github.com/dancannon/gorethink"
)

const dbVersion = 17

var (
	// Address of the RethinkDB cluster instance to connect to
	Address = "localhost:28015"

	// DBName is the name of the database to use
	DBName = "meguca"

	// IsTest can be overridden to not launch several infinite loops during tests
	// or check DB version
	IsTest bool

	// RSession exports the RethinkDB connection session. Used globally by the
	// entire server.
	RSession *r.Session

	// AllTables are all tables needed for meguca operation
	AllTables = []string{
		// Various global information
		"main",

		// Thread data
		"threads",

		// Post data
		"posts",

		// Thumbnailed upload data
		"images",

		// Tokens for claiming thumbnailed images from the "images" table
		"imageTokens",

		// Registered user accounts
		"accounts",

		// Board configurations
		"boards",
	}

	// Map of simple secondary indices for tables
	secondaryIndices = [...]struct {
		table, index string
	}{
		{"threads", "board"},
		{"posts", "op"},
		{"posts", "board"},
		{"posts", "editing"},
		{"posts", "lastUpdated"},
	}

	// Query that increments the database version
	incrementVersion = GetMain("info").Update(map[string]r.Term{
		"dbVersion": r.Row.Field("dbVersion").Add(1),
	})
)

// Document is a generic RethinkDB Document. For DRY-ness.
type Document struct {
	ID string `gorethink:"id"`
}

// Central global information document
type infoDocument struct {
	Document
	DBVersion int `gorethink:"dbVersion"`

	// Is incremented on each new post. Ensures post number uniqueness
	PostCtr int64 `gorethink:"postCtr"`
}

// ConfigDocument holds the global server configurations
type ConfigDocument struct {
	Document
	config.Configs
}

// LoadDB establishes connections to RethinkDB and Redis and bootstraps both
// databases, if not yet done.
func LoadDB() (err error) {
	if err := Connect(); err != nil {
		return err
	}

	var isCreated bool
	err = One(r.DBList().Contains(DBName), &isCreated)
	if err != nil {
		return util.WrapError("error checking, if database exists", err)
	}
	if isCreated {
		RSession.Use(DBName)
		if !IsTest {
			if err := verifyDBVersion(); err != nil {
				return err
			}
		}
	} else if err := InitDB(); err != nil {
		return err
	}

	if !IsTest {
		go runCleanupTasks()
	}
	if err := loadConfigs(); err != nil {
		return err
	}
	return loadBoardConfigs()
}

// Connect establishes a connection to RethinkDB. Address passed separately for
// easier testing.
func Connect() (err error) {
	RSession, err = r.Connect(r.ConnectOpts{Address: Address})
	if err != nil {
		err = util.WrapError("error connecting to RethinkDB", err)
	}
	return
}

// Confirm database version is compatible, if not refuse to start, so we don't
// mess up the DB irreversibly.
func verifyDBVersion() error {
	var version int
	err := One(GetMain("info").Field("dbVersion"), &version)
	if err != nil {
		return util.WrapError("error reading database version", err)
	}

	switch version {
	case dbVersion:
		return nil
	case 14:
		if err := upgrade14to15(); err != nil {
			return err
		}
		fallthrough
	case 15:
		if err := upgrade15to16(); err != nil {
			return err
		}
		fallthrough
	case 16:
		err := WriteAll([]r.Term{
			r.Table("posts").IndexCreate("lastUpdated"),
			incrementVersion,
		})
		if err != nil {
			return err
		}
		if err := waitForIndex("posts")(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("incompatible database version: %d", version)
	}
	return nil
}

// Perform database upgrade from version 14 to 15. Inserts faux creation dates
//  into all board documents.
func upgrade14to15() error {
	qs := [...]r.Term{
		r.Table("boards").Update(map[string]r.Term{
			"created": r.Now(),
		}),
		incrementVersion,
	}

	for _, q := range qs {
		if err := Write(q); err != nil {
			return err
		}
	}
	return nil
}

// Upgrade from version 15 to 16. Contains major structural changes to post
// storage.
func upgrade15to16() error {
	q := r.Table("threads").Config().Update(map[string]string{
		"name": "threads_old",
	})
	if err := Write(q); err != nil {
		return err
	}

	qs := make([]r.Term, 0, 5)
	qs = append(qs, createPostTables()...)
	qs = append(qs,
		// Copy all threads
		r.
			Table("threads").
			Insert(r.Table("threads_old").Without("log", "posts")),
		// Copy all posts
		r.
			Table("threads_old").
			ForEach(func(t r.Term) r.Term {
				return t.
					Field("posts").
					Values().
					Map(func(p r.Term) r.Term {
						return p.Merge(map[string]interface{}{
							"op":          t.Field("id"),
							"board":       t.Field("board"),
							"lastUpdated": time.Now().Unix() - 60,
							"log":         [][]byte{},
						})
					}).
					ForEach(func(p r.Term) r.Term {
						return r.Table("posts").Insert(p)
					})
			}),
		// Delete old table
		r.TableDrop("threads_old"),
		incrementVersion,
	)
	if err := WriteAll(qs); err != nil {
		return err
	}

	return CreateIndices()
}

// InitDB initialize a rethinkDB database
func InitDB() error {
	log.Printf("initializing database '%s'", DBName)
	if err := Write(r.DBCreate(DBName)); err != nil {
		return util.WrapError("creating database", err)
	}

	RSession.Use(DBName)

	if err := CreateTables(); err != nil {
		return err
	}

	return populateDB()
}

// Populate DB with initial documents
func populateDB() error {
	main := [...]interface{}{
		infoDocument{Document{"info"}, dbVersion, 0},

		// History aka progress counters of boards, that get incremented on
		// post and thread creation
		Document{"boardCtrs"},

		ConfigDocument{
			Document{"config"},
			config.Defaults,
		},
	}
	if err := Insert("main", main); err != nil {
		return util.WrapError("initializing database", err)
	}

	if err := CreateAdminAccount(); err != nil {
		return err
	}

	return CreateIndices()
}

// CreateTables creates all tables needed for meguca operation
func CreateTables() error {
	qs := make([]r.Term, 0, len(AllTables))

	for _, t := range AllTables {
		switch t {
		case "images", "threads", "posts":
		default:
			qs = append(qs, createTable(t))
		}
	}

	qs = append(qs, createPostTables()...)
	qs = append(qs, r.TableCreate("images", r.TableCreateOpts{
		PrimaryKey: "SHA1",
	}))

	return WriteAll(qs)
}

func createPostTables() []r.Term {
	fns := make([]r.Term, 2)
	for i, t := range [...]string{"threads", "posts"} {
		fns[i] = r.TableCreate(t, r.TableCreateOpts{
			Durability: "soft",
		})
	}
	return fns
}

func createTable(t string) r.Term {
	return r.TableCreate(t)
}

// CreateIndices creates secondary indices for faster table queries
func CreateIndices() error {
	fns := make([]func() error, 0, len(secondaryIndices)+len(AllTables))

	for _, i := range secondaryIndices {
		index := i // Capture variable
		fns = append(fns, func() error {
			return Write(r.Table(index.table).IndexCreate(index.index))
		})
	}

	// Make sure all indices are ready to avoid the race condition of and index
	// being accessed before its full creation.
	for _, table := range AllTables {
		fns = append(fns, waitForIndex(table))
	}

	return util.Waterfall(fns)
}

func waitForIndex(table string) func() error {
	return func() error {
		return Exec(r.Table(table).IndexWait())
	}
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
