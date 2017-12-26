package db

import "database/sql"

// Thread is a template for writing new threads to the database
type Thread struct {
	ID                  uint64
	PostCtr, ImageCtr   uint32
	ReplyTime, BumpTime int64
	Subject, Board      string
}

// ThreadCounter retrieves the progress counter of a thread
func ThreadCounter(id uint64) (uint64, error) {
	return getCounter("thread_counter", id)
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
	imgCtr := 0
	if p.Image != nil {
		imgCtr = 1
	}

	_, err = getStatement(tx, "insert_thread").Exec(
		append(
			[]interface{}{subject, nonLive, imgCtr},
			genPostCreationArgs(p)...,
		)...,
	)
	if err != nil {
		return
	}

	if p.Editing {
		err = SetOpenBody(p.ID, []byte(p.Body))
	}

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
		t.PostCtr,
		t.ImageCtr,
		t.ReplyTime,
		t.BumpTime,
		t.Subject,
	)
	if err != nil {
		return err
	}

	err = WritePost(tx, p)
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
