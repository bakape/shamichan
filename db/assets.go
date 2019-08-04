package db

import (
	"database/sql"
	"time"

	"github.com/bakape/meguca/assets"
	"github.com/bakape/pg_util"
)

// Load all assets from and pass them to load. Start listening for changes.
func loadAssets(
	table string,
	load func(board string, files []assets.File),
) (err error) {
	byBoard := make(map[string][]assets.File, 64)
	err = queryAll(
		sq.Select("board", "data", "mime").From(table),
		func(r *sql.Rows) (err error) {
			var (
				board string
				file  assets.File
			)
			err = r.Scan(&board, &file.Data, &file.Mime)
			if err != nil {
				return
			}
			byBoard[board] = append(byBoard[board], file)
			return
		},
	)
	if err != nil {
		return
	}

	for board, files := range byBoard {
		load(board, files)
	}

	return Listen(pg_util.ListenOpts{
		Channel:          table + ".updated",
		DebounceInterval: time.Second,
		OnMsg:            updateAssets(table, load),
	})
}

// Returns function for reading assets from db on board asset updates.
// Not inlined to ease testing.
func updateAssets(table string,
	load func(board string, files []assets.File),
) func(string) error {
	return func(board string) (err error) {
		files := make([]assets.File, 0, 16)
		err = queryAll(
			sq.Select("data", "mime").
				From(table).
				Where("board  = ?", board),
			func(r *sql.Rows) (err error) {
				var (
					data []byte
					mime string
				)
				err = r.Scan(&data, &mime)
				if err != nil {
					return
				}
				files = append(files, assets.File{
					Data: data,
					Mime: mime,
				})
				return
			},
		)
		if err != nil {
			return
		}

		load(board, files)
		return
	}
}

func loadBanners() error {
	return loadAssets("banners", assets.Banners.Set)
}

func loadLoadingAnimations() error {
	return loadAssets("loading_animations", setLoadingAnimation)
}

// Outlined to ease testing
func setLoadingAnimation(board string, files []assets.File) {
	var f assets.File
	if len(files) != 0 {
		f = files[0]
	}
	assets.Loading.Set(board, f)
}

// Overwrite any existing assets in the DB
func setAssets(table, board string, files []assets.File) error {
	return InTransaction(func(tx *pgx.Tx) (err error) {
		_, err = sq.
			Delete(table).
			Where("board = ?", board).
			RunWith(tx).
			Exec()
		if err != nil {
			return
		}

		if len(files) != 0 {
			q := sq.
				Insert(table).
				Columns("board", "data", "mime").
				RunWith(tx)
			for _, f := range files {
				if f.Data != nil {
					q = q.Values(board, f.Data, f.Mime)
				}
			}
			_, err = q.Exec()
			if err != nil {
				return
			}
		}

		_, err = tx.Exec("select pg_notify($1 || '.updated', $2)", table, board)
		return
	})
}

// SetBanners overwrites the list of banners in the DB, for a specific board
func SetBanners(board string, banners []assets.File) error {
	return setAssets("banners", board, banners)
}

// SetLoadingAnimation sets the loading animation for a specific board.
// Nil file.Data means the default animation should be used.
func SetLoadingAnimation(board string, file assets.File) error {
	var files []assets.File
	if file.Data != nil {
		files = append(files, file)
	}
	return setAssets("loading_animations", board, files)
}
