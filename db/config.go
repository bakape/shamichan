// Configuration loading, reloading and setting

package db

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/templates"
	"github.com/bakape/meguca/util"
	"github.com/lib/pq"
)

type rowScanner interface {
	Scan(dest ...interface{}) error
}

// Load configs from the database and update on each change
func loadConfigs() error {
	var enc string
	err := DB.QueryRow(`SELECT val FROM main WHERE id = 'config'`).Scan(&enc)
	if err != nil {
		return err
	}
	if err := decodeAndSetConfigs(enc); err != nil {
		return err
	}

	return listen("config_updates", updateConfigs)
}

// Assigns a function to listen to Postgres notifications
func listen(event string, fn func(msg string) error) error {
	l := pq.NewListener(
		connArgs(),
		time.Second,
		time.Second*10,
		func(_ pq.ListenerEventType, _ error) {},
	)
	if err := l.Listen(event); err != nil {
		return err
	}

	go func() {
		if IsTest {
			return
		}
		for msg := range l.Notify {
			if msg.Extra == "" {
				continue
			}
			if err := fn(msg.Extra); err != nil {
				log.Println(err)
			}
		}
	}()

	return nil
}

func decodeAndSetConfigs(data string) error {
	var conf config.Configs
	if err := json.Unmarshal([]byte(data), &conf); err != nil {
		return err
	}
	return config.Set(conf)
}

func loadBoardConfigs() error {
	r, err := DB.Query(`SELECT * FROM boards`)
	if err != nil {
		return err
	}
	defer r.Close()

	for r.Next() {
		c, err := scanBoardConfigs(r)
		if err != nil {
			return err
		}
		if _, err := config.SetBoardConfigs(c); err != nil {
			return err
		}
	}
	if err := r.Err(); err != nil {
		return err
	}

	return listen("board_updated", updateBoardConfigs)
}

func scanBoardConfigs(r rowScanner) (c config.BoardConfigs, err error) {
	var eightball pq.StringArray
	err = r.Scan(
		&c.ReadOnly, &c.TextOnly, &c.ForcedAnon, &c.HashCommands, &c.ID,
		&c.CodeTags, &c.Title, &c.Notice, &c.Rules, &eightball,
	)
	c.Eightball = []string(eightball)
	return
}

// WriteBoardConfigs writes board-specific configurations to the database
func WriteBoardConfigs(c config.BoardConfigs, overwrite bool) error {
	q :=
		`INSERT INTO boards (
			readOnly, textOnly, forcedAnon, hashCommands, id, codeTags, title,
			notice, rules, eightball
		)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	if overwrite {
		q += ` ON CONFLICT DO UPDATE`
	}
	_, err := DB.Exec(q,
		c.ReadOnly, c.TextOnly, c.ForcedAnon, c.HashCommands, c.ID, c.CodeTags,
		c.Title, c.Notice, c.Rules, pq.StringArray(c.Eightball),
	)
	return err
}

// WriteStaff writes staff positions of a specific board. Old rows are
// overwritten.
func WriteStaff(board string, staff map[string][]string) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}

	// Remove previous staff entries
	_, err = tx.Exec(`DELETE FROM TABLE WHERE id = $1`, board)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Write new ones
	p, err := tx.Prepare(
		`INSERT INTO staff (board, account, position) VALUES
			($1, $2, $3)`,
	)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer p.Close()

	for pos, accounts := range staff {
		for _, a := range accounts {
			_, err = p.Exec(board, a, pos)
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit()
}

func updateConfigs(data string) error {
	if err := decodeAndSetConfigs(data); err != nil {
		return util.WrapError("reloading configuration", err)
	}
	return recompileTemplates()
}

func updateBoardConfigs(board string) error {
	r := DB.QueryRow(`SELECT * FROM boards WHERE id = $1`, board)
	conf, err := scanBoardConfigs(r)
	switch err {
	case nil:
	case sql.ErrNoRows:
		config.RemoveBoard(board)
		return recompileTemplates()
	default:
		return err
	}

	changed, err := config.SetBoardConfigs(conf)
	switch {
	case err != nil:
		return util.WrapError("reloading board configuration", err)
	case changed:
		return recompileTemplates()
	default:
		return nil
	}
}

func recompileTemplates() error {
	if IsTest {
		return nil
	}
	if err := templates.Compile(); err != nil {
		return util.WrapError("recompiling templates", err)
	}
	return nil
}
