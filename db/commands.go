package db

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/Chiiruno/meguca/common"
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

// GetPcount retrieves the board's pyu counter
func GetPcount(b string) (c uint64, err error) {
	err = sq.Select("pcount").
		From("pyu").
		Where("id = ?", b).
		Scan(&c)

	return
}

// GetPcountA retrieves the board's pyu counter atomically
func GetPcountA(tx *sql.Tx, b string) (c uint64, err error) {
	err = sq.Select("pcount").
		From("pyu").
		Where("id = ?", b).
		RunWith(tx).
		QueryRow().
		Scan(&c)
	return
}

// IncrementPcount increments the board's pyu counter by one and returns the new counter
func IncrementPcount(tx *sql.Tx, b string) (c uint64, err error) {
	pcount, err := GetPcountA(tx, b)
	if err != nil {
		return
	}

	err = sq.Update("pyu").
		Set("pcount", pcount+1).
		Where("id = ?", b).
		Suffix("returning pcount").
		RunWith(tx).
		QueryRow().
		Scan(&c)
	return
}

// WritePyuLimit creates a new pyu limit row. Only used on the first post of a new IP.
func WritePyuLimit(tx *sql.Tx, ip string, b string) error {
	_, err := sq.Insert("pyu_limit").
		Columns("ip", "board", "restricted", "pcount").
		Values(ip, b, false, 4).
		RunWith(tx).
		Exec()
	return err
}

// PyuLimitExists checks whether an IP has a pyu limit counter
func PyuLimitExists(tx *sql.Tx, ip string, b string) (e bool, err error) {
	err = sq.Select("count(1)").
		From("pyu_limit").
		Where("ip = ? and board = ?", ip, b).
		RunWith(tx).
		QueryRow().
		Scan(&e)
	return
}

// GetPyuLimit retrieves the IP and respective board's pyu limit counter
func GetPyuLimit(tx *sql.Tx, ip string, b string) (c uint8, err error) {
	err = sq.Select("pcount").
		From("pyu_limit").
		Where("ip = ? and board = ?", ip, b).
		RunWith(tx).
		QueryRow().
		Scan(&c)
	return
}

// GetPyuLimitRestricted retrieves the IP and respective board's pyu limit restricted status
func GetPyuLimitRestricted(tx *sql.Tx, ip string, b string,
) (restricted bool, err error) {
	err = sq.Select("restricted").
		From("pyu_limit").
		Where("ip = ? and board = ?", ip, b).
		RunWith(tx).
		QueryRow().
		Scan(&restricted)
	return
}

// SetPyuLimitRestricted sets the IP and respective board's pyu limit restricted status
func SetPyuLimitRestricted(tx *sql.Tx, ip string, b string) (err error) {
	_, err = sq.Update("pyu_limit").
		Set("restricted", true).
		Where("ip = ? and board = ?", ip, b).
		RunWith(tx).
		Exec()
	return
}

// DecrementPyuLimit decrements the pyu limit counter by one and returns the new counter
func DecrementPyuLimit(tx *sql.Tx, ip string, b string) (err error) {
	pcount, err := GetPyuLimit(tx, ip, b)
	if err != nil {
		return
	}

	_, err = sq.Update("pyu_limit").
		Set("pcount", pcount-1).
		Where("ip = ? and board = ?", ip, b).
		RunWith(tx).
		Exec()
	return
}

// FreePyuLimit resets the restricted status and pcount so sluts can #pyu again.
func FreePyuLimit() error {
	_, err := sq.Update("pyu_limit").
		SetMap(map[string]interface{}{"restricted": false, "pcount": 4}).
		Exec()
	return err
}
