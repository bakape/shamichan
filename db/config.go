package db

import (
	"encoding/json"
	"time"

	"github.com/bakape/meguca/assets"
	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	mlog "github.com/bakape/meguca/log"
	"github.com/bakape/meguca/util"
	"github.com/bakape/pg_util"
	"github.com/jackc/pgx"
)

// BoardConfigs contains extra fields not exposed on database reads
type BoardConfigs struct {
	config.BoardConfigs
	Created time.Time
}

type rowScanner interface {
	Scan(...interface{}) error
}

// Load configs from the database and update on each change
func loadConfigs() error {
	conf, err := GetConfigs()
	if err != nil {
		return err
	}
	config.Set(conf)
	mlog.Init(mlog.Email)

	return Listen(pg_util.ListenOpts{
		Channel: "configs.updated",
		OnMsg: func(_ string) error {
			conf, err := GetConfigs()
			if err != nil {
				return util.WrapError("reloading configuration", err)
			}
			config.Set(conf)
			mlog.Update()

			return auth.LoadCaptchaServices()
		},
	})
}

// GetConfigs retrieves global configurations. Only used in tests.
func GetConfigs() (c config.Configs, err error) {
	err = db.QueryRow("get_configs").Scan(&c)
	return
}

func loadBoardConfigs() (err error) {
	r, err := db.Query("get_all_board_configs")
	if err != nil {
		return
	}
	defer r.Close()

	var c config.BoardConfigs
	for r.Next() {
		c, err = scanBoardConfigs(r)
		if err != nil {
			return
		}
		_, err = config.SetBoardConfigs(c)
		if err != nil {
			return
		}
	}
	err = r.Err()
	if err != nil {
		return
	}

	return Listen(pg_util.ListenOpts{
		DebounceInterval: time.Second,
		Channel:          "boards.updated",
		OnMsg:            updateBoardConfigs,
	})
}

func scanBoardConfigs(r rowScanner) (c config.BoardConfigs, err error) {
	err = r.Scan(
		&c.ID,
		&c.ReadOnly,
		&c.TextOnly,
		&c.ForcedAnon,
		&c.DisableRobots,
		&c.Flags,
		&c.NSFW,
		&c.RbText,
		&c.Pyu,
		&c.DefaultCSS,
		&c.Title,
		&c.Notice,
		&c.Rules,
		&c.Eightball,
	)
	c.Banners = assets.Banners.FileTypes(c.ID)
	return
}

// WriteBoard writes a board complete with configurations to the database
func WriteBoard(tx *pgx.Tx, c BoardConfigs) (err error) {
	if c.Created.IsZero() {
		c.Created = time.Now().UTC()
	}
	_, err = tx.Exec(
		"insert_board",
		c.ID,
		c.ReadOnly,
		c.TextOnly,
		c.ForcedAnon,
		c.DisableRobots,
		c.Flags,
		c.NSFW,
		c.RbText,
		c.Pyu,
		c.Created,
		c.DefaultCSS,
		c.Title,
		c.Notice,
		c.Rules,
		c.Eightball,
	)
	return
}

// UpdateBoard updates board configurations
func UpdateBoard(c config.BoardConfigs) (err error) {
	_, err = db.Exec(
		"update_board",
		c.ID,
		c.ReadOnly,
		c.TextOnly,
		c.ForcedAnon,
		c.DisableRobots,
		c.Flags,
		c.NSFW,
		c.RbText,
		c.Pyu,
		c.DefaultCSS,
		c.Title,
		c.Notice,
		c.Rules,
		c.Eightball,
	)
	return
}

// Separated for easier unit testing
func updateBoardConfigs(board string) error {
	conf, err := GetBoardConfigs(board)
	switch err {
	case nil:
	case pgx.ErrNoRows:
		config.RemoveBoard(board)
		return nil
	default:
		return err
	}

	changed, err := config.SetBoardConfigs(conf)
	switch {
	case err != nil:
		return util.WrapError("reloading board configuration", err)
	case changed:
		return auth.LoadCaptchaServices()
	default:
		return nil
	}
}

// GetBoardConfigs retrives the configurations of a specific board
func GetBoardConfigs(board string) (config.BoardConfigs, error) {
	return scanBoardConfigs(db.QueryRow("get_board_configs", board))
}

// WriteConfigs writes new global configurations to the database
func WriteConfigs(c config.Configs) (err error) {
	data, err := json.Marshal(c)
	if err != nil {
		return
	}
	_, err = db.Exec(
		`update main
		set val = $1
		where key = 'config'`,
		data,
	)
	return
}
