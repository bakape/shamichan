package db

import (
	"database/sql"
	"encoding/json"
	"meguca/config"
	"meguca/templates"
	"meguca/util"
	"time"

	"github.com/lib/pq"
)

// BoardConfigs contains extra fields not exposed on database reads
type BoardConfigs struct {
	config.BoardConfigs
	Created time.Time
}

// Load configs from the database and update on each change
func loadConfigs() error {
	conf, err := GetConfigs()
	if err != nil {
		return err
	}
	config.Set(conf)

	return Listen("config_updates", updateConfigs)
}

// GetConfigs retrieves global configurations. Only used in tests.
func GetConfigs() (c config.Configs, err error) {
	var enc string
	err = db.QueryRow(`SELECT val FROM main WHERE id = 'config'`).Scan(&enc)
	if err != nil {
		return
	}
	c, err = decodeConfigs(enc)
	return
}

func decodeConfigs(data string) (c config.Configs, err error) {
	err = json.Unmarshal([]byte(data), &c)
	return
}

func loadBoardConfigs() error {
	r, err := prepared["get_all_board_configs"].Query()
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

	return Listen("board_updated", updateBoardConfigs)
}

func scanBoardConfigs(r rowScanner) (c config.BoardConfigs, err error) {
	var eightball pq.StringArray
	err = r.Scan(
		&c.ReadOnly, &c.TextOnly, &c.ForcedAnon, &c.DisableRobots, &c.Flags,
		&c.NSFW, &c.NonLive, &c.PosterIDs,
		&c.ID, &c.DefaultCSS, &c.Title, &c.Notice, &c.Rules, &eightball, &c.Js,
	)
	c.Eightball = []string(eightball)
	return
}

// WriteBoard writes a board complete with configurations to the database
func WriteBoard(tx *sql.Tx, c BoardConfigs) error {
	_, err := getStatement(tx, "write_board").Exec(
		c.ID, c.ReadOnly, c.TextOnly, c.ForcedAnon, c.DisableRobots, c.Flags,
		c.NSFW, c.NonLive, c.PosterIDs,
		c.Created, c.DefaultCSS, c.Title, c.Notice, c.Rules,
		pq.StringArray(c.Eightball), c.Js,
	)
	return err
}

// UpdateBoard updates board configurations
func UpdateBoard(c config.BoardConfigs) error {
	return execPrepared(
		"update_board",
		c.ID, c.ReadOnly, c.TextOnly, c.ForcedAnon, c.DisableRobots, c.Flags,
		c.NSFW, c.NonLive, c.PosterIDs,
		c.DefaultCSS, c.Title, c.Notice, c.Rules,
		pq.StringArray(c.Eightball), c.Js,
	)
}

func updateConfigs(data string) error {
	conf, err := decodeConfigs(data)
	if err != nil {
		return util.WrapError("reloading configuration", err)
	}
	config.Set(conf)

	return recompileTemplates()
}

func updateBoardConfigs(board string) error {
	conf, err := GetBoardConfigs(board)
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

// GetBoardConfigs retrives the configurations of a specific board
func GetBoardConfigs(board string) (config.BoardConfigs, error) {
	return scanBoardConfigs(prepared["get_board_configs"].QueryRow(board))
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

// WriteConfigs writes new global configurations to the database
func WriteConfigs(c config.Configs) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return execPrepared("write_configs", string(data))
}
