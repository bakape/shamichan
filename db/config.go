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

	// Load and set initial board slice. Technically there is a data race here.
	// If a board is delted by another backend instance in the time between this
	// query and loadBoardConfigs() sees the update, this backend instance will
	// not see the update. But considering how much compilcation working around
	// this would bring, let's ingonre this for now.
	q := r.Table("boards").Field("id").CoerceTo("array")
	var boards []string
	if err := All(q, &boards); err != nil {
		return err
	}
	config.SetBoards(boards)

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
					"id":      b.Field("id"),
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
	if err := config.SetBoardConfigs(u.BoardConfigs); err != nil {
		return util.WrapError("reloading board configuration", err)
	}
	return nil
}

func recompileTemplates() error {
	if err := templates.Compile(); err != nil {
		return util.WrapError("recompiling teplates", err)
	}
	return nil
}
