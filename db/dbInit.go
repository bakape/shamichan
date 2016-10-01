// Initialises and loads RethinkDB

package db

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/util"
	r "github.com/dancannon/gorethink"
)

const dbVersion = 16

var (
	// Address of the RethinkDB cluster instance to connect to
	Address = "localhost:28015"

	// DBName is the name of the database to use
	DBName = "meguca"

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

	// Map of simple secondary indececes for tables
	secondaryIndeces = [...]struct {
		table, index string
	}{
		{"threads", "board"},
		{"posts", "op"},
		{"posts", "board"},
		{"posts", "editing"},
	}
)

// Document is a eneric RethinkDB Document. For DRY-ness.
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
		if !isTest {
			if err := verifyDBVersion(); err != nil {
				return err
			}
		}
	} else if err := InitDB(); err != nil {
		return err
	}

	if !isTest {
		go runCleanupTasks()
	}
	return loadConfigs()
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

// Confirm database verion is compatible, if not refuse to start, so we don't
// mess up the DB irreversably.
func verifyDBVersion() error {
	var version int
	err := One(GetMain("info").Field("dbVersion"), &version)
	if err != nil {
		return util.WrapError("error reading database version", err)
	}
	if version != dbVersion {
		if version == 14 {
			if err := upgrade14to15(); err != nil {
				return err
			}
		} else {
			return fmt.Errorf(
				"incompatible RethinkDB database version: %d",
				version,
			)
		}
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
		r.Table("main").Get("info").Update(map[string]int{
			"dbVersion": 15,
		}),
	}

	for _, q := range qs {
		if err := Write(q); err != nil {
			return err
		}
	}
	return nil
}

// InitDB initialize a rethinkDB database
func InitDB() error {
	log.Printf("initialising database '%s'", DBName)
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

	if err := createAdminAccount(); err != nil {
		return err
	}

	return CreateIndeces()
}

// CreateTables creates all tables needed for meguca operation
func CreateTables() error {
	fns := make([]func() error, 0, len(AllTables))

	for _, table := range AllTables {
		switch table {
		case "images", "threads", "posts":
		default:
			fns = append(fns, createTable(table))
		}
	}

	fns = append(fns, func() error {
		return Write(r.TableCreate("images", r.TableCreateOpts{
			PrimaryKey: "SHA1",
		}))
	})

	softDurability := [...]string{"threads", "posts"}
	for i := range softDurability {
		table := softDurability[i]
		fns = append(fns, func() error {
			return Write(r.TableCreate(table, r.TableCreateOpts{
				Durability: "soft",
			}))
		})
	}

	return util.Waterfall(fns)
}

func createTable(name string) func() error {
	return func() error {
		return Write(r.TableCreate(name))
	}
}

// CreateIndeces create secondary indeces for faster table queries
func CreateIndeces() error {
	fns := make([]func() error, 0, len(secondaryIndeces)+len(AllTables))

	for _, i := range secondaryIndeces {
		index := i // Capture variable
		fns = append(fns, func() error {
			return Write(r.Table(index.table).IndexCreate(index.index))
		})
	}

	// Make sure all indeces are ready to avoid the race condition of and index
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

// UniqueDBName returns a unique datatabase name. Needed so multiple concurent
// `go test` don't clash in the same database.
func UniqueDBName() string {
	return "meguca_tests_" + strconv.FormatInt(time.Now().UnixNano(), 10)
}

// Create the admin account and write it to the database
func createAdminAccount() error {
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
