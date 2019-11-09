package db

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bakape/meguca/common"
	"github.com/bakape/pg_util"
	"github.com/go-playground/log"
	"github.com/jackc/pgx"
	"github.com/lib/pq"
)

// InTransaction runs a function inside a transaction and handles comminting and
// rollback on error.
func InTransaction(fn func(*pgx.Tx) error) (err error) {
	return pg_util.InTransaction(db, fn)
}

// IsConflictError returns if an error is a unique key conflict error
func IsConflictError(err error) bool {
	return pqErrorCode(err) == "unique_violation"
}

// Extract error code, if error is a *pq.Error
func pqErrorCode(err error) string {
	if err, ok := err.(*pq.Error); ok {
		return err.Code.Name()
	}
	return ""
}

func logListenError(err error) {
	log.Error(err)
}

// Listen assigns a function to listen to Postgres notifications on a channel.
func Listen(opts pg_util.ListenOpts) (err error) {
	// Don't allow non-cancellable listeners to run  during tests
	if common.IsTest && opts.Canceller == nil {
		return
	}

	opts.ConnectionURL = connectionURL
	opts.OnError = logListenError
	return pg_util.Listen(opts)
}

// PostgreSQL notification message parse error
type ErrMsgParse string

func (e ErrMsgParse) Error() string {
	return fmt.Sprintf("unparsable message: `%s`", string(e))
}

// Split message containing a board and a variable amount of int64
func SplitBoardAndInts(msg string, intCount int) (
	board string,
	ints []int64,
	err error,
) {
	split := strings.Split(msg, ",")
	if len(split) != intCount+1 {
		goto fail
	}
	board = split[0]

	for _, s := range split[1:] {
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			goto fail
		}
		ints = append(ints, n)
	}
	return

fail:
	err = ErrMsgParse(msg)
	return
}

// Split message containing a list of uint64 numbers.
// Returns error, if message did not contain n integers.
func SplitUint64s(msg string, n int) (arr []uint64, err error) {
	parts := strings.Split(msg, ",")
	if len(parts) != n {
		goto fail
	}
	for _, p := range parts {
		i, err := strconv.ParseUint(p, 10, 64)
		if err != nil {
			goto fail
		}
		arr = append(arr, i)
	}
	return

fail:
	err = ErrMsgParse(msg)
	return
}

// Try to extract an exception message, if err is *pq.Error
func extractException(err error) string {
	if err, ok := err.(*pgx.PgError); ok {
		return err.Message
	}
	return ""
}

// Encode []uint64 tom postgres format
func encodeUint64Array(arr []uint64) string {
	b := []byte{'{'}
	for i, j := range arr {
		if i != 0 {
			b = append(b, ',')
		}
		b = strconv.AppendUint(b, j, 10)
	}
	return string(append(b, '}'))
}

type idSorter []uint64

func (p idSorter) Len() int           { return len(p) }
func (p idSorter) Less(i, j int) bool { return p[i] < p[j] }
func (p idSorter) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
