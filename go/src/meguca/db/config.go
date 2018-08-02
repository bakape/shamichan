package db

import (
	"database/sql"
	"encoding/json"
	"meguca/assets"
	"meguca/config"
	"meguca/log"
	"meguca/templates"
	"meguca/util"
	"time"

	"github.com/Masterminds/squirrel"
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
	mLog.Init(mLog.Email)

	return Listen("config_updates", updateConfigs)
}

// GetConfigs retrieves global configurations. Only used in tests.
func GetConfigs() (c config.Configs, err error) {
	var enc string
	err = sq.Select("val").
		From("main").
		Where("id = 'config'").
		QueryRow().
		Scan(&enc)
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

func getBoardConfigs() squirrel.SelectBuilder {
	return sq.Select(
		"readOnly", "textOnly", "forcedAnon", "disableRobots", "flags", "NSFW",
		"nonLive", "posterIDs", "rbText", "pyu", "id", "defaultCSS", "title", "notice", "rules",
		"eightball",
	).
		From("boards")
}

func loadBoardConfigs() error {
	r, err := getBoardConfigs().Query()
	if err != nil {
		return err
	}
	defer r.Close()

	for r.Next() {
		c, err := scanBoardConfigs(r)
		if err != nil {
			return err
		}
		c.Banners = assets.Banners.FileTypes(c.ID)
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
		&c.NSFW, &c.NonLive, &c.PosterIDs, &c.RbText, &c.Pyu,
		&c.ID, &c.DefaultCSS, &c.Title, &c.Notice, &c.Rules, &eightball,
	)
	c.Eightball = []string(eightball)
	return
}

// WriteBoard writes a board complete with configurations to the database
func WriteBoard(tx *sql.Tx, c BoardConfigs) error {
	q := sq.Insert("boards").
		Columns(
			"id", "readOnly", "textOnly", "forcedAnon", "disableRobots",
			"flags", "NSFW", "nonLive",
			"posterIDs", "rbText", "pyu", "created", "defaultCSS", "title", "notice",
			"rules", "eightball",
		).
		Values(
			c.ID, c.ReadOnly, c.TextOnly, c.ForcedAnon, c.DisableRobots,
			c.Flags, c.NSFW, c.NonLive, c.PosterIDs, c.RbText, c.Pyu,
			c.Created, c.DefaultCSS, c.Title, c.Notice, c.Rules,
			pq.StringArray(c.Eightball),
		)

	err := withTransaction(tx, q).Exec()

	if err != nil {
		return err
	}

	return notifyBoardUpdated(tx, c.ID)
}

func notifyBoardUpdated(tx *sql.Tx, board string) error {
	_, err := tx.Exec("select pg_notify('board_updated', $1)", board)
	return err
}

// UpdateBoard updates board configurations
func UpdateBoard(c config.BoardConfigs) error {
	return InTransaction(false, func(tx *sql.Tx) error {
		q := sq.Update("boards").
			SetMap(map[string]interface{}{
				"readOnly":      c.ReadOnly,
				"textOnly":      c.TextOnly,
				"forcedAnon":    c.ForcedAnon,
				"disableRobots": c.DisableRobots,
				"flags":         c.Flags,
				"NSFW":          c.NSFW,
				"nonLive":       c.NonLive,
				"posterIDs":     c.PosterIDs,
				"rbText":        c.RbText,
				"pyu":           c.Pyu,
				"defaultCSS":    c.DefaultCSS,
				"title":         c.Title,
				"notice":        c.Notice,
				"rules":         c.Rules,
				"eightball":     pq.StringArray(c.Eightball),
			}).
			Where("id = ?", c.ID)
		err := withTransaction(tx, q).Exec()
		if err != nil {
			return err
		}
		return notifyBoardUpdated(tx, c.ID)
	})
}

func updateConfigs(data string) error {
	conf, err := decodeConfigs(data)
	if err != nil {
		return util.WrapError("reloading configuration", err)
	}
	config.Set(conf)
	mLog.Update()

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

	// Inject banners into configuration struct
	conf.Banners = assets.Banners.FileTypes(board)

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
	q := getBoardConfigs().Where("id = ?", board)
	return scanBoardConfigs(q.QueryRow())
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
func WriteConfigs(c config.Configs) (err error) {
	data, err := json.Marshal(c)
	if err != nil {
		return
	}
	s := string(data)
	_, err = sq.Update("main").
		Set("val", s).
		Where("id = 'config'").
		Exec()
	if err != nil {
		return
	}
	_, err = db.Exec("select pg_notify('config_updates', $1)", s)
	return
}
