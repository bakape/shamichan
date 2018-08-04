package db

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"meguca/common"

	"github.com/lib/pq"
	"github.com/mailru/easyjson"
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
		s, err := easyjson.Marshal(c[i])
		if err != nil {
			return nil, err
		}
		strArr[i] = string(s)
	}

	return strArr.Value()
}

// WritePyu creates a new board's pyu row. Only used on board creation
func WritePyu(b string) error {
	_, err := sq.Insert("pyu").
		Columns("id", "pcount").
		Values(b, 0).
		Exec()

	return err
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
	r, err := withTransaction(tx, sq.Select("pcount").
		From("pyu").
		Where("id = ?", b)).
		QueryRow()

	if err != nil {
		return
	}

	err = r.Scan(&c)
	return
}

// IncrementPcount increments the board's pyu counter by one and returns the new counter
func IncrementPcount(tx *sql.Tx, b string) (c uint64, err error) {
	pcount, err := GetPcountA(tx, b)

	if err != nil {
		return
	}

	r, err := withTransaction(tx, sq.Update("pyu").
		Set("pcount", pcount+1).
		Where("id = ?", b).
		Suffix("returning pcount")).
		QueryRow()

	if err != nil {
		return
	}

	err = r.Scan(&c)
	return
}

// SetPcount sets the board's pyu counter. Only used in tests.
func SetPcount(c uint64) error {
	_, err := sq.Update("pyu").
		Set("pcount", c).
		Exec()

	return err
}

// WritePyuLimit creates a new pyu limit row. Only used on the first post of a new IP.
func WritePyuLimit(tx *sql.Tx, ip string, b string) error {
	return withTransaction(tx, sq.Insert("pyu_limit").
		Columns("ip", "board", "restricted", "pcount").
		Values(ip, b, false, 4)).
		Exec()
}

// PyuLimitExists checks whether an IP has a pyu limit counter
func PyuLimitExists(tx *sql.Tx, ip string, b string) (e bool, err error) {
	r, err := withTransaction(tx, sq.Select("count(1)").
		From("pyu_limit").
		Where("ip = ? and board = ?", ip, b)).
		QueryRow()

	if err != nil {
		return
	}

	err = r.Scan(&e)
	return
}

// GetPyuLimit retrieves the IP and respective board's pyu limit counter
func GetPyuLimit(tx *sql.Tx, ip string, b string) (c uint8, err error) {
	r, err := withTransaction(tx, sq.Select("pcount").
		From("pyu_limit").
		Where("ip = ? and board = ?", ip, b)).
		QueryRow()

	if err != nil {
		return
	}

	err = r.Scan(&c)
	return
}

// GetPyuLimitRestricted retrieves the IP and respective board's pyu limit restricted status
func GetPyuLimitRestricted(tx *sql.Tx, ip string, b string) (restricted bool, err error) {
	r, err := withTransaction(tx, sq.Select("restricted").
		From("pyu_limit").
		Where("ip = ? and board = ?", ip, b)).
		QueryRow()

	if err != nil {
		return
	}

	err = r.Scan(&restricted)
	return
}

// SetPyuLimitRestricted sets the IP and respective board's pyu limit restricted status
func SetPyuLimitRestricted(tx *sql.Tx, ip string, b string) error {
	return withTransaction(tx, sq.Update("pyu_limit").
		Set("restricted", true).
		Where("ip = ? and board = ?", ip, b)).
		Exec()
}

// DecrementPyuLimit decrements the pyu limit counter by one and returns the new counter
func DecrementPyuLimit(tx *sql.Tx, ip string, b string) error {
	pcount, err := GetPyuLimit(tx, ip, b)

	if err != nil {
		return err
	}

	return withTransaction(tx, sq.Update("pyu_limit").
		Set("pcount", pcount-1).
		Where("ip = ? and board = ?", ip, b)).
		Exec()
}

// FreePyuLimit resets the restricted status and pcount so sluts can #pyu again.
func FreePyuLimit() error {
	_, err := sq.Update("pyu_limit").
		SetMap(map[string]interface{}{"restricted":false, "pcount":4}).
		Exec()
	return err
}

// DecrementRoulette retrieves current roulette counter and decrements it
func DecrementRoulette() (c uint8, err error) {
	err = db.QueryRow(`
		update main
			set val = (val::smallint - 1)::text
			where id = 'roulette'
			returning val::smallint + 1`).
		Scan(&c)
	return
}

// ResetRoulette resets the roulette counter to 6
func ResetRoulette() (err error) {
	_, err = sq.Update("main").
		Set("val", "6").
		Where(`id = 'roulette'`).
		Exec()
	return
}

// GetRcount retrieves current roulette counter
func GetRcount() (c uint64, err error) {
	err = sq.Select("val::bigint").
		From("main").
		Where("id = 'rcount'").
		QueryRow().
		Scan(&c)
	return
}

// IncrementRcount increments the roulette counter by one
func IncrementRcount() (err error) {
	_, err = db.Exec(`
		update main
			set val = (val::bigint + 1)::text
			where id = 'rcount'`)
	return
}
