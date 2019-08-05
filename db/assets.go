package db

import (
	"fmt"
	"time"

	"github.com/bakape/meguca/assets"
	"github.com/bakape/pg_util"
	"github.com/jackc/pgx"
)

// Load all assets from and pass them to load. Start listening for changes.
func loadAssets(
	table string,
	load func(board string, files []assets.File),
) (err error) {
	byBoard := make(map[string][]assets.File)

	r, err := db.Query(`select board, data, mime from ` + table)
	if err != nil {
		return
	}
	defer r.Close()

	var (
		board string
		file  assets.File
	)
	for r.Next() {
		// Avoid sharing the same buffer
		file.Data = nil

		err = r.Scan(&board, &file.Data, &file.Mime)
		if err != nil {
			return
		}

		byBoard[board] = append(byBoard[board], file)
	}
	err = r.Err()
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
func updateAssets(
	table string,
	load func(board string, files []assets.File),
) func(string) error {
	return func(board string) (err error) {
		files := make([]assets.File, 0, 16)

		r, err := db.Query(
			fmt.Sprintf(
				`select data, mime
				from %s
				where board = $1`,
				table,
			),
			board,
		)
		if err != nil {
			return
		}
		defer r.Close()

		var (
			data []byte
			mime string
		)
		for r.Next() {
			data = nil // Avoid sharing buffer

			err = r.Scan(&data, &mime)
			if err != nil {
				return
			}
			files = append(files, assets.File{
				Data: data,
				Mime: mime,
			})
		}
		err = r.Err()
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
		_, err = tx.Exec(
			fmt.Sprintf(`delete from %s where board = $1`, table),
			board,
		)
		if err != nil {
			return
		}

		if len(files) != 0 {
			name := "insert_" + table
			_, err = tx.Prepare(
				name,
				fmt.Sprintf(
					`insert into %s (board, data, mime)
					values ($1, $2, $3)`,
				),
			)
			if err != nil {
				return
			}

			for _, f := range files {
				if f.Data == nil {
					continue
				}
				_, err = tx.Exec(name, board, f.Data, f.Mime)
				if err != nil {
					return
				}
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
