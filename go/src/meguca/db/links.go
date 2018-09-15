package db

import (
	"bytes"
	"database/sql"
	"fmt"
	"meguca/common"
	"strconv"
)

// Decodes post links from Postgres array aggregations
type linkScanner []common.Link

func (l *linkScanner) Scan(src interface{}) error {
	switch src := src.(type) {
	case []byte:
		return l.scanBytes(src)
	case string:
		return l.scanBytes([]byte(src))
	case nil:
		*l = nil
		return nil
	default:
		return fmt.Errorf("cannot convert %T to []common.Link", src)
	}
}

func (l *linkScanner) scanBytes(src []byte) (err error) {
	// Determine needed size and preallocate final array
	n := 0
	for _, b := range src {
		if b == '(' {
			n++
		}
	}
	*l = make(linkScanner, 0, n)

	var (
		start = 0
		link  common.Link
	)
	for i, b := range src {
		switch b {
		case '(':
			start = i + 1
		case ')':
			split := bytes.Split(src[start:i], []byte{','})
			if len(split) != 3 {
				return fmt.Errorf(
					"invalid tuple structure: `%s` at range [%d:%d]",
					string(src), start, i)
			}
			link.ID, err = strconv.ParseUint(string(split[0]), 10, 64)
			if err != nil {
				return
			}
			link.OP, err = strconv.ParseUint(string(split[1]), 10, 64)
			if err != nil {
				return
			}
			link.Board = string(split[2])

			*l = append(*l, link)
		}
	}

	return
}

// Write post links to database
func writeLinks(tx *sql.Tx, source uint64, links []common.Link) (err error) {
	q, err := tx.Prepare(
		`insert into links (source, target)
		values($1, $2)
		on conflict do nothing`)
	if err != nil {
		return
	}

	// Need to deduplicate to prevent primary key collisions
	written := make(map[uint64]bool, len(links))
	for _, l := range links {
		if written[l.ID] {
			continue
		}
		_, err = q.Exec(source, l.ID)
		if err != nil {
			return
		}
		written[l.ID] = true
	}
	return
}
