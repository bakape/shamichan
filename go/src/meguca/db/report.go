package db

import "meguca/auth"

// Report a post for rule violations
func Report(id uint64, board, reason, ip string, illegal bool) error {
	return execPrepared("report", id, board, reason, ip, illegal)
}

// Read reports for a specific board. Pass "all" for global reports.
func GetReports(board string) (rep []auth.Report, err error) {
	r, err := prepared["get_reports"].Query(board)
	if err != nil {
		return
	}
	defer r.Close()

	var tmp auth.Report
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
