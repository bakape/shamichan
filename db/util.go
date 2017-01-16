package db

import "database/sql"

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

func setReadOnly(tx *sql.Tx) error {
	_, err := tx.Exec("SET TRANSACTION READ ONLY")
	return err
}
