package db

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/bakape/meguca/assets"
	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	mlog "github.com/bakape/meguca/log"
	"github.com/bakape/meguca/templates"
	"github.com/bakape/meguca/util"
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
	mlog.Init(mlog.Email)

	return Listen("config_updates", updateConfigs)
}

// GetConfigs retrieves global configurations. Only used in tests.
func GetConfigs() (c config.Configs, err error) {
	var enc []byte
	err = sq.Select("val").
		From("main").
		Where("id = 'config'").
		QueryRow().
		Scan(&enc)
	if err != nil {
		return
	}
	err = json.Unmarshal(enc, &c)
	return
}

func getBoardConfigs() squirrel.SelectBuilder {
	return sq.Select(
		"readOnly", "textOnly", "forcedAnon", "disableRobots", "flags", "NSFW",
		"rbText", "pyu", "id", "defaultCSS", "title", "notice",
		"rules", "eightball",
	).
		From("boards")
}

func loadBoardConfigs() (err error) {
	err = queryAll(getBoardConfigs(), func(r *sql.Rows) (err error) {
		c, err := scanBoardConfigs(r)
		if err != nil {
			return
		}
		c.Banners = assets.Banners.FileTypes(c.ID)
		_, err = config.SetBoardConfigs(c)
		return
	})
	if err != nil {
		return
	}
	return Listen("board_updated", updateBoardConfigs)
}

func scanBoardConfigs(r rowScanner) (c config.BoardConfigs, err error) {
	var eightball pq.StringArray
	err = r.Scan(
		&c.ReadOnly, &c.TextOnly, &c.ForcedAnon, &c.DisableRobots, &c.Flags,
		&c.NSFW, &c.RbText, &c.Pyu,
		&c.ID, &c.DefaultCSS, &c.Title, &c.Notice, &c.Rules, &eightball,
	)
	c.Eightball = []string(eightball)
	return
}

// WriteBoard writes a board complete with configurations to the database
func WriteBoard(tx *sql.Tx, c BoardConfigs) error {
	_, err := sq.Insert("boards").
		Columns(
			"id", "readOnly", "textOnly", "forcedAnon", "disableRobots",
			"flags", "NSFW",
			"rbText", "pyu", "created", "defaultCSS", "title",
			"notice", "rules", "eightball",
		).
		Values(
			c.ID, c.ReadOnly, c.TextOnly, c.ForcedAnon, c.DisableRobots,
			c.Flags, c.NSFW, c.RbText, c.Pyu,
			c.Created, c.DefaultCSS, c.Title, c.Notice, c.Rules,
			pq.StringArray(c.Eightball),
		).
		RunWith(tx).
		Exec()
	return err
}

// UpdateBoard updates board configurations
func UpdateBoard(c config.BoardConfigs) (err error) {
	_, err = sq.Update("boards").
		SetMap(map[string]interface{}{
			"readOnly":      c.ReadOnly,
			"textOnly":      c.TextOnly,
			"forcedAnon":    c.ForcedAnon,
			"disableRobots": c.DisableRobots,
			"flags":         c.Flags,
			"NSFW":          c.NSFW,
			"rbText":        c.RbText,
			"pyu":           c.Pyu,
			"defaultCSS":    c.DefaultCSS,
			"title":         c.Title,
			"notice":        c.Notice,
			"rules":         c.Rules,
			"eightball":     pq.StringArray(c.Eightball),
		}).
		Where("id = ?", c.ID).
		Exec()
	return
}

func updateConfigs(_ string) error {
	conf, err := GetConfigs()
	if err != nil {
		return util.WrapError("reloading configuration", err)
	}
	config.Set(conf)
	mlog.Update()

	return util.Parallel(templates.Recompile, auth.LoadCaptchaServices)
}

func updateBoardConfigs(board string) error {
	conf, err := GetBoardConfigs(board)
	switch err {
	case nil:
	case sql.ErrNoRows:
		config.RemoveBoard(board)
		return templates.Recompile()
	default:
		return err
	}

	// Inject banners into configuration struct
	conf.Banners = assets.Banners.FileTypes(board)

	changed, err := config.SetBoardConfigs(conf)
	switch {
	case err != nil:
		return util.WrapError("reloading board configuration", err)
	case changed:
		return util.Parallel(templates.Recompile, auth.LoadCaptchaServices)
	default:
		return nil
	}
}

// GetBoardConfigs retrives the configurations of a specific board
func GetBoardConfigs(board string) (config.BoardConfigs, error) {
	q := getBoardConfigs().Where("id = ?", board)
	return scanBoardConfigs(q.QueryRow())
}

// WriteConfigs writes new global configurations to the database
func WriteConfigs(c config.Configs) (err error) {
	data, err := json.Marshal(c)
	if err != nil {
		return
	}
	_, err = sq.Update("main").
		Set("val", string(data)).
		Where("id = 'config'").
		Exec()
	if err != nil {
		return
	}
	_, err = db.Exec("select pg_notify('config_updates', '')")
	return
}
