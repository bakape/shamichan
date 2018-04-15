package db

import (
	"database/sql"
	"time"
)

// Thread is a template for writing new threads to the database
type Thread struct {
	ID                  uint64
	PostCtr, ImageCtr   uint32
	ReplyTime, BumpTime int64
	Subject, Board      string
}

// ThreadCounter retrieves the progress counter of a thread
func ThreadCounter(id uint64) (uint64, error) {
	q := sq.Select("replyTime").
		From("threads").
		Where("id = ?", id)
	return getCounter(q)
}

// ValidateOP confirms the specified thread exists on specific board
func ValidateOP(id uint64, board string) (valid bool, err error) {
	err = prepared["validate_op"].QueryRow(id, board).Scan(&valid)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return
}

// InsertThread inserts a new thread into the database.
func InsertThread(tx *sql.Tx, subject string, nonLive bool, p Post) (
	err error,
) {
	err = withTransaction(tx,
		sq.Insert("threads").
			Columns(
				"board", "id", "replyTime", "bumpTime", "subject", "nonLive",
			).
			Values(p.Board, p.ID, p.Time, p.Time, subject, nonLive),
	).Exec()
	if err != nil {
		return
	}

	err = WritePost(tx, p, false, false)
	return
}

// WriteThread writes a thread and it's OP to the database. Only used for tests
// and migrations.
func WriteThread(tx *sql.Tx, t Thread, p Post) (err error) {
	passedTx := tx != nil
	if !passedTx {
		tx, err = db.Begin()
		if err != nil {
			return err
		}
		defer RollbackOnError(tx, &err)
	}

	_, err = tx.Stmt(prepared["write_op"]).Exec(
		t.Board,
		t.ID,
		t.ReplyTime,
		t.BumpTime,
		t.Subject,
	)
	if err != nil {
		return err
	}

	err = WritePost(tx, p, false, false)
	if err != nil {
		return err
	}

	if !passedTx {
		return tx.Commit()
	}
	return nil
}

// Check, if a thread has live post updates disabled
func CheckThreadNonLive(id uint64) (nonLive bool, err error) {
	return queryBool(id, "check_thread_nonlive")
}

// Perform a query by id that returns a boolean
func queryBool(id uint64, queryID string) (val bool, err error) {
	err = prepared[queryID].QueryRow(id).Scan(&val)
	return
}

// Check, if a thread has been locked by a moderator
func CheckThreadLocked(id uint64) (bool, error) {
	return queryBool(id, "check_thread_locked")
}

// Increment thread update, bump, post and image counters
func bumpThread(tx *sql.Tx, id uint64, bump bool) (err error) {
	now := time.Now().Unix()
	q := sq.Update("threads").Set("replyTime", now)

	if bump {
		var (
			r         rowScanner
			postCount uint
		)
		r, err = withTransaction(tx,
			sq.Select("count(*)").
				From("posts").
				Where("op = ?", id),
		).
			QueryRow()
		if err != nil {
			return
		}
		err = r.Scan(&postCount)
		if err != nil {
			return
		}

		if postCount < 3000 {
			q = q.Set("bumpTime", now)
		}
	}

	err = withTransaction(tx, q.Where("id = ?", id)).Exec()
	return err
}
