package db

import (
	"database/sql"

	"github.com/lib/pq"
)

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

type queryer interface {
	Exec(string, ...interface{}) (sql.Result, error)
	Query(string, ...interface{}) (*sql.Rows, error)
	QueryRow(string, ...interface{}) *sql.Row
}

// StartTransaction initiates a new DB transaction. It is the responsibility of
// the caller to commit or rollback the transaction.
func StartTransaction() (*sql.Tx, error) {
	return db.Begin()
}

// GetPyu retrieves current pyu counter
func GetPyu() (c int, err error) {
	err = db.QueryRow(`SELECT val::bigint FROM main WHERE id = 'pyu'`).Scan(&c)
	return
}

// IncrementPyu increments the pyu counter by one and returns the new counter
func IncrementPyu() (c int, err error) {
	const q = `
		UPDATE main
			SET val = (val::bigint + 1)::text
			WHERE id = 'pyu'
			RETURNING val::bigint`
	err = db.QueryRow(q).Scan(&c)
	return
}

// SetPyu sets the pyu counter
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

func getQuerier(tx *sql.Tx) queryer {
	if tx == nil {
		return db
	}
	return tx
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
