package db

import (
	"database/sql"
	"meguca/auth"
)

// Report a post for rule violations
func Report(id uint64, board, reason, ip string, illegal bool) error {
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
