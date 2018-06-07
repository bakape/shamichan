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

// WritePyu creates a new board's pyu row. Only used on board creation
func WritePyu(b string) (err error) {
	_, err = sq.Insert("pyu").
		Columns("id", "pcount").
		Values(b, 0).
		Exec()

	return
}

// GetPcount retrieves the board's pyu counter
func GetPcount(b string) (c uint64, err error) {
	err = sq.Select("pcount").
		From("pyu").
		Where("id = ?", b).
		Scan(&c)

	return
}

// GetPcount retrieves the board's pyu counter atomically
func GetPcountA(tx *sql.Tx, b string) (c uint64, err error) {
	r, err := withTransaction(tx, sq.Select("pcount").
		From("pyu").
		Where("id = ?", b)).
		QueryRow()

	if err != nil {
		return
	}

	err = r.Scan(&c)
	return
}

// IncrementPcount increments the board's pyu counter by one and returns the new counter
func IncrementPcount(tx *sql.Tx, b string) (c uint64, err error) {
	pcount, err := GetPcountA(tx, b)

	if err != nil {
		return
	}

	r, err := withTransaction(tx, sq.Update("pyu").
		Set("pcount", pcount + 1).
		Where("id = ?", b).
		Suffix("returning pcount")).
		QueryRow()

	if err != nil {
		return
	}

	err = r.Scan(&c)
	return
}

// SetPcount sets the board's pyu counter. Only used in tests.
func SetPcount(c uint64) (err error) {
	_, err = sq.Update("pyu").
		Set("pcount", c).
		Exec()

	return err
}

// WritePyuLimit creates a new pyu limit row. Only used on the first post of a new IP.
func WritePyuLimit(tx *sql.Tx, ip string, b string) error {
	return withTransaction(tx, sq.Insert("pyu_limit").
		Columns("ip", "board", "expires", "pcount").
		Values(ip, b, time.Now().Add(-time.Second).UTC(), 4)).
		Exec()
}

// PyuLimitExists checks whether an IP has a pyu limit counter
func PyuLimitExists(tx *sql.Tx, ip string, b string) (e bool, err error) {
	r, err := withTransaction(tx, sq.Select("count(1)").
		From("pyu_limit").
		Where("ip = ? and board = ?", ip, b)).
		QueryRow()

	if err != nil {
		return
	}

	err = r.Scan(&e)
	return
}

// GetPyuLimit retrieves the IP and respective board's pyu limit counter
func GetPyuLimit(tx *sql.Tx, ip string, b string) (c uint8, err error) {
	r, err := withTransaction(tx, sq.Select("pcount").
		From("pyu_limit").
		Where("ip = ? and board = ?", ip, b)).
		QueryRow()

	if err != nil {
		return
	}

	err = r.Scan(&c)
	return
}

// GetPyuLimitExpire retrieves the IP and respective board's pyu limit expire timestamp
func GetPyuLimitExpires(tx *sql.Tx, ip string, b string) (t time.Time, err error) {
	r, err := withTransaction(tx, sq.Select("expires").
		From("pyu_limit").
		Where("ip = ? and board = ?", ip, b)).
		QueryRow()

	if err != nil {
		return
	}

	err = r.Scan(&t)
	return
}

// SetPyuLimitExpire sets the IP and respective board's pyu limit expire timestamp
func SetPyuLimitExpires(tx *sql.Tx, ip string, b string) error {
	return withTransaction(tx, sq.Update("pyu_limit").
		Set("expires", time.Now().Add(time.Hour).UTC()).
		Where("ip = ? and board = ?", ip, b)).
		Exec()
}

// DecrementPyuLimit decrements the pyu limit counter by one and returns the new counter
func DecrementPyuLimit(tx *sql.Tx, ip string, b string) (err error) {
	pcount, err := GetPyuLimit(tx, ip, b)

	if err != nil {
		return
	}

	return withTransaction(tx, sq.Update("pyu_limit").
		Set("pcount", pcount - 1).
		Where("ip = ? and board = ?", ip, b)).
		Exec()
}

// ResetPyuLimit resets the pyu limit counter to 4
func ResetPyuLimit(tx *sql.Tx, ip string, b string) (c uint8, err error) {
	r, err := withTransaction(tx, sq.Update("pyu_limit").
		Set("pcount", 4).
		Where("ip = ? and board = ?", ip, b).
		Suffix("returning pcount")).
		QueryRow()

	if err != nil {
		return
	}

	err = r.Scan(&c)
	return
}

// DecrementRoulette retrieves current roulette counter and decrements it
func DecrementRoulette() (c uint8, err error) {
	err = db.QueryRow(`
		update main
			set val = (val::smallint - 1)::text
			where id = 'roulette'
			returning val::smallint + 1`).
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
	_, err = db.Exec(`
		update main
			set val = (val::bigint + 1)::text
			where id = 'rcount'`)
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
