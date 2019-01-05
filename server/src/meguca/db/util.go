package db

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/go-playground/log"
	"github.com/lib/pq"
)

type rowScanner interface {
	Scan(dest ...interface{}) error
}

// InTransaction runs a function inside a transaction and handles comminting and rollback on error.
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
func queryAll(q squirrel.SelectBuilder, fn func(r *sql.Rows) error,
) (err error) {
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

// Listen assigns a function to listen to Postgres notifications on a channel
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
				log.Errorf(
					"error on database event id=`%s` msg=`%s` error=`%s`\n",
					event, msg.Extra, err)
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

// Split message containing a board and post/thread ID
func SplitBoardAndID(msg string) (board string, id uint64, err error) {
	setErr := func() {
		err = fmt.Errorf("unparsable message: '%s'", msg)
	}

	split := strings.Split(msg, ",")
	if len(split) != 2 {
		setErr()
		return
	}
	board = split[0]
	id, err = strconv.ParseUint(split[1], 10, 64)
	if err != nil {
		setErr()
	}
	return
}
