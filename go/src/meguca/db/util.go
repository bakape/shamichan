package db

import (
	"context"
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

type queryer interface {
	Query() (*sql.Rows, error)
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

func (t transactionalQuery) Query() (r *sql.Rows, err error) {
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
// error.
// readOnly: the DBMS can optimise read-only transactions for better concurrency
func InTransaction(readOnly bool, fn func(*sql.Tx) error) (err error) {
	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{
		ReadOnly: readOnly,
	})
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

// Run fn on all returned rows in a query
func queryAll(q queryer, fn func(r *sql.Rows) error) (err error) {
	r, err := q.Query()
	if err != nil {
		return
	}
	defer r.Close()

	for r.Next() {
		err = fn(r)
		if err != nil {
			return
		}
	}
	return r.Err()
}

// IsConflictError returns if an error is a unique key conflict error
func IsConflictError(err error) bool {
	if err, ok := err.(*pq.Error); ok && err.Code.Name() == "unique_violation" {
		return true
	}
	return false
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

// GetGeoMD5 retrieves the GeoIP MD5 hash
func GetGeoMD5() (hash string, err error) {
	err = sq.Select("val::char(32)").
		From("main").
		Where("id = 'geo_md5'").
		Scan(&hash)

	return
}

// SetGeoMD5 sets the GeoIP MD5 hash
func SetGeoMD5(hash string) error {
	_, err := sq.Update("main").
		Set("val", hash).
		Where("id = 'geo_md5'").
		Exec()
		
	return err
}
