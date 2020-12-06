package db

import (
	"context"
	"log"

	"github.com/bakape/meguca/imager/config"
	"github.com/bakape/pg_util"
)

// Load configurations from the database and continue loading any updates
func loadConfig(ctx context.Context) (err error) {
	ch := make(chan struct{}, 1)
	err = pg_util.Listen(pg_util.ListenOpts{
		ConnectionURL: connectionURL,
		Channel:       "config.updated",
		OnMsg: func(msg string) error {
			select {
			case <-ctx.Done():
			case ch <- struct{}{}:
			}
			return nil
		},
		OnError: func(err error) {
			log.Printf("config: error listening for config updates: %s\n", err)
		},
		OnReconnect: func() {
			select {
			case <-ctx.Done():
			case ch <- struct{}{}:
			}
		},
		Context: ctx,
	})
	if err != nil {
		return
	}

	read := func() (err error) {
		var c config.Config
		err = db.
			QueryRow(ctx, `select val from main where key = 'config'`).
			Scan(&c)
		if err != nil {
			return
		}
		config.Set(c)
		return
	}

	err = read()
	if err != nil {
		return
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ch:
				err := read()
				if err != nil {
					log.Printf(
						"config: error reading config updates: %s\n",
						err,
					)
				}
			}
		}
	}()

	return
}
