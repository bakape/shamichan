package db

import "meguca/auth"

// Report a post for rule violations
func Report(id uint64, board, reason, ip string, illegal bool) error {
	_, err := sq.Insert("reports").
		Columns("target", "board", "reason", "by", "illegal").
		Values(id, board, reason, ip, illegal).
		Exec()
	return err
}

// Read reports for a specific board. Pass "all" for global reports.
func GetReports(board string) (rep []auth.Report, err error) {
	r, err := sq.Select("id", "target", "reason", "created").
		From("reports").
		Where("board = ?", board).
		OrderBy("created desc").
		Query()
	if err != nil {
		return
	}
	defer r.Close()

	tmp := auth.Report{
		Board: board,
	}
	rep = make([]auth.Report, 0, 64)
	for r.Next() {
		err = r.Scan(&tmp.ID, &tmp.Target, &tmp.Reason, &tmp.Created)
		if err != nil {
			return
		}
		rep = append(rep, tmp)
	}

	err = r.Err()
	return
}
