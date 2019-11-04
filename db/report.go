package db

import (
	"database/sql"
	"fmt"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/go-playground/log"
)

// Report a post for rule violations
func Report(id uint64, board, reason, ip string, illegal bool) error {
	// If the reported content is illegal, log an error so it will email
	if illegal {
		log.Errorf(
			"Illegal content reported\nPost: %s/all/%d\nReason: %s\nIP: %s",
			config.Get().RootURL, id, reason, ip)
	}

	_, err := sq.Insert("reports").
		Columns("target", "board", "reason", "by", "illegal").
		Values(id, board, reason, ip, illegal).
		Exec()

	return err
}

// GetReports reads reports for a specific board.
// Pass "all" for global reports.
func GetReports(board string) (rep []auth.Report, err error) {
	var where string
	rep = make([]auth.Report, 0, 64)
	tmp := auth.Report{Board: board}

	if board == "all" {
		where = "illegal = true"
	} else {
		where = fmt.Sprintf("board = '%s'", board)
	}

	err = queryAll(
		sq.Select("id", "target", "reason", "created").
			From("reports").
			Where(where).
			OrderBy("created desc"),
		func(r *sql.Rows) (err error) {
			err = r.Scan(&tmp.ID, &tmp.Target, &tmp.Reason, &tmp.Created)

			if err != nil {
				return
			}

			rep = append(rep, tmp)
			return
		},
	)

	return
}
