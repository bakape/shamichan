package db

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"errors"
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

// For decoding and encoding the tuple arrays we used to store links in.
// Still needed for migrations.
type linkRowLegacy [][2]uint64

func (l *linkRowLegacy) Scan(src interface{}) error {
	switch src := src.(type) {
	case []byte:
		return l.scanBytes(src)
	case string:
		return l.scanBytes([]byte(src))
	case nil:
		*l = nil
		return nil
	default:
		return fmt.Errorf("cannot convert %T to [][2]uint", src)
	}
}

func (l *linkRowLegacy) scanBytes(src []byte) error {
	length := len(src)
	if length < 6 {
		return errors.New("source too short")
	}

	src = src[1 : length-1]

	// Determine needed size and preallocate final array
	commas := 0
	for _, b := range src {
		if b == ',' {
			commas++
		}
	}
	*l = make(linkRowLegacy, 0, (commas-1)/2+1)

	var (
		inner bool
		next  [2]uint64
		err   error
		buf   = make([]byte, 0, 16)
	)
	for _, b := range src {
		switch b {
		case '{': // New tuple
			inner = true
			buf = buf[0:0]
		case ',':
			if inner { // End of first uint
				next[0], err = strconv.ParseUint(string(buf), 10, 64)
				if err != nil {
					return err
				}
				buf = buf[0:0]
			}
		case '}': // End of tuple
			next[1], err = strconv.ParseUint(string(buf), 10, 64)
			if err != nil {
				return err
			}
			*l = append(*l, next)
		default:
			buf = append(buf, b)
		}
	}

	return nil
}

func (l linkRowLegacy) Value() (driver.Value, error) {
	n := len(l)
	if n == 0 {
		return nil, nil
	}

	b := make([]byte, 1, 16)
	b[0] = '{'
	for i, l := range l {
		if i != 0 {
			b = append(b, ',')
		}
		b = append(b, '{')
		b = strconv.AppendUint(b, l[0], 10)
		b = append(b, ',')
		b = strconv.AppendUint(b, l[1], 10)
		b = append(b, '}')
	}
	b = append(b, '}')

	return string(b), nil
}

// Write post links to database
func writeLinks(tx *sql.Tx, source uint64, links []common.Link) (err error) {
	q, err := tx.Prepare(`insert into links (source, target) values($1, $2)`)
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
