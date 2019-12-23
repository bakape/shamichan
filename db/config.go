package db

import (
	"encoding/json"
	"fmt"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	mlog "github.com/bakape/meguca/log"
	"github.com/bakape/pg_util"
)

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
				return fmt.Errorf("reloading configuration: %w", err)
			}
			config.Set(conf)
			mlog.Update()

			return auth.LoadCaptchaServices()
		},
	})
}

// GetConfigs retrieves global configurations. Only used in tests.
func GetConfigs() (c config.Configs, err error) {
	err = db.
		QueryRow(
			`select val
			from main
			where key = 'config'`,
		).
		Scan(&c)
	return
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
