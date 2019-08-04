package db

import (
	"database/sql"
	"fmt"
)

var (
	triggerExecTimeStrings = [...]string{"before", "instead of", "after"}
	tableEventStrings      = [...]string{"insert", "update", "delete"}
)

// Time of trigger execution, relative to query triggering the trigger
type triggerExecTime uint8

const (
	before triggerExecTime = iota
	insteadOf
	after
)

func (t triggerExecTime) String() string {
	return triggerExecTimeStrings[int(t)]
}

// Database table modification event
type tableEvent uint8

const (
	tableInsert tableEvent = iota
	tableUpdate
	tableDelete
)

func (t tableEvent) String() string {
	return tableEventStrings[int(t)]
}

type triggerDescriptor struct {
	execTime triggerExecTime
	source   tableEvent
}

// Register triggers and trigger functions for each board in triggers
func registerTriggers(tx *pgx.Tx, triggers map[string][]triggerDescriptor,
) (err error) {
	for table, desc := range triggers {
		err = loadSQL(tx, "triggers/"+table)
		if err != nil {
			return
		}
		for _, d := range desc {
			name := fmt.Sprintf(`%s_%s_%s`, d.execTime, table, d.source)

			_, err = tx.Exec(fmt.Sprintf(`drop trigger if exists %s on %s`,
				name, table))
			if err != nil {
				return
			}

			_, err = tx.Exec(fmt.Sprintf(
				`create trigger %s
					%s %s on %s
					for each row
					execute procedure %s()`,
				name, d.execTime, d.source, table, name))
			if err != nil {
				return
			}
		}
	}
	return
}
