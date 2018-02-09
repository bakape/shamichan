package db

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"meguca/common"
	"strconv"

	"github.com/lib/pq"
)

// For decoding and encoding the tuple arrays we used to store links in.
// Still needed for migrations.
type linkRow [][2]uint64

func (l *linkRow) Scan(src interface{}) error {
	switch src := src.(type) {
	case []byte:
		return l.scanBytes(src)
	case string:
		return l.scanBytes([]byte(src))
	case nil:
		*l = nil
		return nil
	default:
		return fmt.Errorf("db: cannot convert %T to [][2]uint", src)
	}
}

func (l *linkRow) scanBytes(src []byte) error {
	length := len(src)
	if length < 6 {
		return errors.New("db: source too short")
	}

	src = src[1 : length-1]

	// Determine needed size and preallocate final array
	commas := 0
	for _, b := range src {
		if b == ',' {
			commas++
		}
	}
	*l = make(linkRow, 0, (commas-1)/2+1)

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

func (l linkRow) Value() (driver.Value, error) {
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

// Get links originating from posts by id
func getLinks(ids ...uint64) (links map[uint64][]common.Link, err error) {
	arg := make(pq.Int64Array, len(ids))
	for i := 0; i < len(ids); i++ {
		arg[i] = int64(ids[i])
	}

	r, err := prepared["get_links"].Query(arg)
	if err != nil {
		return
	}
	defer r.Close()
	links = make(map[uint64][]common.Link, len(ids))
	var (
		link   common.Link
		source uint64
	)
	for r.Next() {
		err = r.Scan(&source, &link.ID, &link.OP, &link.Board)
		if err != nil {
			return
		}
		links[source] = append(links[source], link)
	}
	err = r.Err()
	if err != nil {
		return
	}

	return
}

// Write post links to database
func writeLinks(tx *sql.Tx, source uint64, links []common.Link) (err error) {
	q := getExecutor(tx, "write_link")
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
