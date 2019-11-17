package db

import (
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
			config.Get().RootURL, id, reason, ip,
		)
	}

	_, err := db.Exec("insert_report", id, board, reason, ip, illegal)
	return err
}

// GetReports reads reports for a specific board. Pass "all" for global reports.
func GetReports(board string) (rep []auth.Report, err error) {
	rep = make([]auth.Report, 0, 64)

	r, err := db.Query(
		`select id, target, reason, created
		from reports
		where board = $1
		order by created desc`,
	)
	if err != nil {
		return
	}
	defer r.Close()

	tmp := auth.Report{
		Board: board,
	}
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
