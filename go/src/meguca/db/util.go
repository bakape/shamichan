//go:generate go-bindata -o bin_data.go --pkg db --nometadata --prefix sql sql/...

package db

import (
	"database/sql"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/lib/pq"
)

// Stores generated prepared statements
var prepared = make(map[string]*sql.Stmt)

type executor interface {
	Exec(args ...interface{}) (sql.Result, error)
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

type tableScanner interface {
	rowScanner
	Next() bool
	Err() error
	Close() error
}

// Generate prepared statements
func genPrepared() error {
	names := AssetNames()
	sort.Strings(names)
	left := make([]string, 0, len(names))

	for _, id := range names {
		switch {
		case strings.HasPrefix(id, "init"):
			continue
		case strings.HasPrefix(id, "functions"):
			_, err := db.Exec(getQuery(id))
			if err != nil {
				return err
			}
		default:
			left = append(left, id)
		}
	}

	for _, id := range left {
		var err error
		k := strings.TrimSuffix(filepath.Base(id), ".sql")
		prepared[k], err = db.Prepare(getQuery(id))
		if err != nil {
			return err
		}
	}

	return nil
}

// StartTransaction initiates a new DB transaction. It is the responsibility of
// the caller to commit or rollback the transaction.
func StartTransaction() (*sql.Tx, error) {
	return db.Begin()
}

// GetPyu retrieves current pyu counter
func GetPyu() (c uint64, err error) {
	err = prepared["get_pyu"].QueryRow().Scan(&c)
	return
}

// IncrementPyu increments the pyu counter by one and returns the new counter
func IncrementPyu() (c uint64, err error) {
	err = prepared["increment_pyu"].QueryRow().Scan(&c)
	return
}

// SetPyu sets the pyu counter. Only used in tests.
func SetPyu(c uint) error {
	_, err := db.Exec(`UPDATE main SET val = $1::text WHERE id = 'pyu'`, c)
	return err
}

func getExecutor(tx *sql.Tx, key string) executor {
	if tx != nil {
		return tx.Stmt(prepared[key])
	}
	return prepared[key]
}

func execPrepared(id string, args ...interface{}) error {
	_, err := prepared[id].Exec(args...)
	return err
}

func getStatement(tx *sql.Tx, id string) *sql.Stmt {
	stmt := prepared[id]
	if tx != nil {
		stmt = tx.Stmt(stmt)
	}
	return stmt
}

func setReadOnly(tx *sql.Tx) error {
	_, err := tx.Exec("SET TRANSACTION READ ONLY")
	return err
}

// IsConflictError returns if an error is a unique key conflict error
func IsConflictError(err error) bool {
	if err, ok := err.(*pq.Error); ok && err.Code.Name() == "unique_violation" {
		return true
	}
	return false
}

// RollbackOnError on error undoes the transaction on error
func RollbackOnError(tx *sql.Tx, err *error) {
	if *err != nil {
		tx.Rollback()
	}
}

// Retrieve binary-encoded SQL query
func getQuery(id string) string {
	return string(MustAsset(id))
}

// Assigns a function to listen to Postgres notifications on a channel
func Listen(event string, fn func(msg string) error) (err error) {
	if IsTest {
		return
	}

	l := pq.NewListener(
		ConnArgs,
		time.Second,
		time.Second*10,
		func(_ pq.ListenerEventType, _ error) {},
	)
	err = l.Listen(event)
	if err != nil {
		return
	}

	go func() {
		for msg := range l.Notify {
			if msg == nil {
				continue
			}
			if err := fn(msg.Extra); err != nil {
				log.Println(err)
			}
		}
	}()

	return
}

// Execute all SQL statement strings and return on first error, if any
func execAll(tx *sql.Tx, q ...string) error {
	for _, q := range q {
		if _, err := tx.Exec(q); err != nil {
			return err
		}
	}
	return nil
}
