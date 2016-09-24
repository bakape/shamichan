// Configuration loading, reloading and setting

package db

import (
	"log"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/templates"
	"github.com/bakape/meguca/util"
	r "github.com/dancannon/gorethink"
)

// Override to not launch several infinite loops during tests
var isTest bool

// Load configs from the database and on each change
func loadConfigs() error {
	cursor, err := GetMain("config").
		Changes(r.ChangesOpts{IncludeInitial: true}).
		Field("new_val").
		Run(RSession)
	if err != nil {
		return err
	}

	read := make(chan config.Configs)
	cursor.Listen(read)
	initial := <-read

	// Reaload configuration on any change in the database
	go func() {
		if isTest {
			return
		}
		for {
			if err := updateConfigs(<-read); err != nil {
				log.Println(err)
			}
		}
	}()

	return config.Set(initial)
}

func updateConfigs(conf config.Configs) error {
	if err := config.Set(conf); err != nil {
		return util.WrapError("reloading configuration", err)
	}
	if err := templates.Compile(); err != nil {
		return util.WrapError("recompiling teplates", err)
	}
	return nil
}
