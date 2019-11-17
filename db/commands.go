package db

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/bakape/meguca/common"
	"github.com/jackc/pgx"
	"github.com/lib/pq"
)

// For encoding and decoding hash command results
type commandRow []common.Command

func (c *commandRow) Scan(src interface{}) error {
	switch src := src.(type) {
	case []byte:
		return c.scanBytes(src)
	case string:
		return c.scanBytes([]byte(src))
	case nil:
		*c = nil
		return nil
	default:
		return fmt.Errorf("cannot convert %T to []common.Command", src)
	}
}

func (c *commandRow) scanBytes(data []byte) (err error) {
	var sArr pq.StringArray
	err = sArr.Scan(data)
	if err != nil {
		return
	}

	*c = make([]common.Command, len(sArr))
	for i := range sArr {
		err = (*c)[i].UnmarshalJSON([]byte(sArr[i]))
		if err != nil {
			return
		}
	}

	return
}

func (c commandRow) Value() (driver.Value, error) {
	if c == nil {
		return nil, nil
	}

	var strArr = make(pq.StringArray, len(c))
	for i := range strArr {
		s, err := json.Marshal(c[i])
		if err != nil {
			return nil, err
		}
		strArr[i] = string(s)
	}

	return strArr.Value()
}

// Populate command results that need DB access. Commands results that don't
// need DB access are assigned in the parser.
func populateCommands(tx *pgx.Tx, com []common.Command) (err error) {
	for i := range com {
		switch com[i].Type {
		case common.Pyu:
			err = tx.
				QueryRow(
					`update main
					set val = val::bigint + 1
					where id = 'pyu'
					returning val::bigint`,
				).
				Scan(&com[i].Pyu)
			if err != nil {
				return
			}
		case common.Pcount:
			err = tx.
				QueryRow(
					`select val::bigint
					from main
					where id = 'pyu'`,
				).
				Scan(&com[i].Pyu)
			if err != nil {
				return
			}
		}
	}
	return
}
