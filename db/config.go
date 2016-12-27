// Configuration loading, reloading and setting

package db

import (
	"encoding/json"
	"log"
	"time"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/templates"
	"github.com/bakape/meguca/util"
	"github.com/lib/pq"
)

// type boardConfUpdate struct {
// 	Deleted bool
// 	config.BoardConfigs
// }

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

	// Listen for updates
	listener := pq.NewListener(
		connArgs(),
		time.Second,
		time.Second*10,
		func(_ pq.ListenerEventType, _ error) {},
	)
	if err := listener.Listen("config_updates"); err != nil {
		return err
	}
	go func() {
		if IsTest {
			return
		}
		for msg := range listener.Notify {
			if msg.Extra == "" {
				continue
			}
			if err := updateConfigs(msg.Extra); err != nil {
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

// func loadBoardConfigs() error {
// 	// First load all boards
// 	var all []config.BoardConfigs
// 	if err := All(r.Table("boards"), &all); err != nil {
// 		return err
// 	}
// 	for _, conf := range all {
// 		_, err := config.SetBoardConfigs(conf)
// 		if err != nil {
// 			return err
// 		}
// 	}

// 	// Then start listening to updates. This will also contain the initial
// 	// values, so we deduplicate server-side.
// 	cursor, err := r.
// 		Table("boards").
// 		Changes(r.ChangesOpts{
// 			IncludeInitial: true,
// 		}).
// 		Map(func(b r.Term) r.Term {
// 			return r.Branch(
// 				b.Field("new_val").Eq(nil),
// 				map[string]interface{}{
// 					"deleted": true,
// 					"id":      b.Field("old_val").Field("id"),
// 				},
// 				b.Field("new_val"),
// 			)
// 		}).
// 		Run(RSession)
// 	if err != nil {
// 		return err
// 	}

// 	read := make(chan boardConfUpdate)
// 	cursor.Listen(read)

// 	if IsTest {
// 		return nil
// 	}
// 	go func() {
// 		for {
// 			if err := updateBoardConfigs(<-read); err != nil {
// 				log.Println(err)
// 			}
// 		}
// 	}()

// 	return nil
// }

func updateConfigs(data string) error {
	if err := decodeAndSetConfigs(data); err != nil {
		return util.WrapError("reloading configuration", err)
	}
	return recompileTemplates()
}

// func updateBoardConfigs(u boardConfUpdate) error {
// 	if u.Deleted {
// 		config.RemoveBoard(u.ID)
// 		return recompileTemplates()
// 	}

// 	changed, err := config.SetBoardConfigs(u.BoardConfigs)
// 	if err != nil {
// 		return util.WrapError("reloading board configuration", err)
// 	}
// 	if changed {
// 		return recompileTemplates()
// 	}
// 	return nil
// }

func recompileTemplates() error {
	if IsTest {
		return nil
	}
	if err := templates.Compile(); err != nil {
		return util.WrapError("recompiling templates", err)
	}
	return nil
}
