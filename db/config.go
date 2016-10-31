// Configuration loading, reloading and setting

package db

import (
	"log"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/templates"
	"github.com/bakape/meguca/util"
	r "github.com/dancannon/gorethink"
)

type boardConfUpdate struct {
	Deleted bool
	config.BoardConfigs
}

// Load configs from the database and update on each change
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

	// Reload configuration on any change in the database
	if !IsTest {
		go func() {
			for {
				if err := updateConfigs(<-read); err != nil {
					log.Println(err)
				}
			}
		}()
	}

	return config.Set(initial)
}

func loadBoardConfigs() error {
	// First load all boards
	var all []config.BoardConfigs
	if err := All(r.Table("boards"), &all); err != nil {
		return err
	}
	for _, conf := range all {
		_, err := config.SetBoardConfigs(conf)
		if err != nil {
			return err
		}
	}

	// Then start listening to updates. This will also contain the initial
	// values, so we deduplicate server-side.
	cursor, err := r.
		Table("boards").
		Changes(r.ChangesOpts{
			IncludeInitial: true,
		}).
		Map(func(b r.Term) r.Term {
			return r.Branch(
				b.Field("new_val").Eq(nil),
				map[string]interface{}{
					"deleted": true,
					"id":      b.Field("old_val").Field("id"),
				},
				b.Field("new_val"),
			)
		}).
		Run(RSession)
	if err != nil {
		return err
	}

	read := make(chan boardConfUpdate)
	cursor.Listen(read)

	if IsTest {
		return nil
	}
	go func() {
		for {
			if err := updateBoardConfigs(<-read); err != nil {
				log.Println(err)
			}
		}
	}()

	return nil
}

func updateConfigs(conf config.Configs) error {
	if err := config.Set(conf); err != nil {
		return util.WrapError("reloading configuration", err)
	}
	return recompileTemplates()
}

func updateBoardConfigs(u boardConfUpdate) error {
	if u.Deleted {
		config.RemoveBoard(u.ID)
		return recompileTemplates()
	}

	changed, err := config.SetBoardConfigs(u.BoardConfigs)
	if err != nil {
		return util.WrapError("reloading board configuration", err)
	}
	if changed {
		return recompileTemplates()
	}
	return nil
}

func recompileTemplates() error {
	if err := templates.Compile(); err != nil {
		return util.WrapError("recompiling templates", err)
	}
	return nil
}
