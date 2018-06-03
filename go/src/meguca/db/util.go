package db

import (
	"database/sql"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/go-playground/log"
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
	Err() error
	Close() error
}

// Allows easily running squirrel queries with transactions
type transactionalQuery struct {
	tx *sql.Tx
	sq squirrel.Sqlizer
}

func withTransaction(tx *sql.Tx, q squirrel.Sqlizer) transactionalQuery {
	return transactionalQuery{
		tx: tx,
		sq: q,
	}
}

func (t transactionalQuery) Exec() (err error) {
	sql, args, err := t.sq.ToSql()
	if err != nil {
		return
	}
	_, err = t.tx.Exec(sql, args...)
	return
}

func (t transactionalQuery) Query() (ts tableScanner, err error) {
	sql, args, err := t.sq.ToSql()
	if err != nil {
		return
	}
	return t.tx.Query(sql, args...)
}

func (t transactionalQuery) QueryRow() (rs rowScanner, err error) {
	sql, args, err := t.sq.ToSql()
	if err != nil {
		return
	}
	rs = t.tx.QueryRow(sql, args...)
	return
}

// Runs function inside a transaction and handles comminting and rollback on
// error
func InTransaction(fn func(*sql.Tx) error) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return
	}

	err = fn(tx)
	if err != nil {
		tx.Rollback()
		return
	}
	return tx.Commit()
}

// StartTransaction initiates a new DB transaction. It is the responsibility of
// the caller to commit or rollback the transaction.
func StartTransaction() (*sql.Tx, error) {
	return db.Begin()
}

// DecrementRoulette retrieves current roulette counter and decrements it
func DecrementRoulette() (c uint8, err error) {
	err = sq.Update("main").
		Set("val", "(val::smallint - 1)::text").
		Where("id = 'roulette'").
		Suffix("returning val::smallint + 1").
		QueryRow().
		Scan(&c)
	return
}

// ResetRoulette resets the roulette counter to 6
func ResetRoulette() (err error) {
	_, err = sq.Update("main").
		Set("val", "6").
		Where(`id = 'roulette'`).
		Exec()
	return
}

// GetRcount retrieves current roulette counter
func GetRcount() (c uint64, err error) {
	err = sq.Select("val::bigint").
		From("main").
		Where("id = 'rcount'").
		QueryRow().
		Scan(&c)
	return
}

// IncrementRcount increments the roulette counter by one
func IncrementRcount() (err error) {
	_, err = sq.Update("main").
		Set("val", "(val::bigint + 1)::text").
		Where("id = 'rcount'").
		Exec()
	return
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
				log.Errorf("error on database event `%s`: %s\n", event, err)
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
