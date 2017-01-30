package db

import (
	"database/sql"
	"path/filepath"
	"strings"

	queries "github.com/bakape/meguca/db/sql"
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
}

// Generate prepared statements
func genPrepared() error {
	for _, id := range queries.AssetNames() {
		if strings.HasPrefix(id, "init") {
			continue
		}

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
func GetPyu() (c int, err error) {
	err = prepared["get_pyu"].QueryRow().Scan(&c)
	return
}

// IncrementPyu increments the pyu counter by one and returns the new counter
func IncrementPyu() (c int, err error) {
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
	b, err := queries.Asset(id)
	if err != nil {
		panic(err)
	}
	return string(b)
}
