package db

import (
	"database/sql"

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

// GetReports reads reports for a specific board. Pass "all" for global reports.
func GetReports(board string) (rep []auth.Report, err error) {
	tmp := auth.Report{
		Board: board,
	}
	rep = make([]auth.Report, 0, 64)
	err = queryAll(
		sq.Select("id", "target", "reason", "created").
			From("reports").
			Where("board = ?", board).
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
