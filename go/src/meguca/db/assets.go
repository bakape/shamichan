package db

import (
	"database/sql"
	"meguca/assets"
	"meguca/common"
)

// Overwrite list of banners in the DB, for a specific board
func SetBanners(board string, banners []assets.File) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return
	}
	defer RollbackOnError(tx, &err)

	_, err = tx.Stmt(prepared["clear_banners"]).Exec(board)
	if err != nil {
		return
	}

	q := tx.Stmt(prepared["set_banner"])
	for i, f := range banners {
		_, err = q.Exec(board, i, f.Data, f.Mime)
		if err != nil {
			return
		}
	}

	_, err = tx.Exec("select pg_notify('banners_updated', $1)", board)
	if err != nil {
		return
	}

	err = tx.Commit()
	return
}

func loadBanners() (err error) {
	// Load all banners and group by board
	byBoard := make(map[string][]assets.File, 64)
	err = loadAllAssets("load_all_banners", func(b string, f assets.File) {
		files := byBoard[b]
		if files == nil {
			files = make([]assets.File, 0, common.MaxNumBanners)
		}
		byBoard[b] = append(files, f)
	})
	if err != nil {
		return
	}

	for board, files := range byBoard {
		assets.Banners.Set(board, files)
	}

	return Listen("banners_updated", updateBanners)
}

// Load all assets by prepared query key and pass them to fn one by one
func loadAllAssets(q string, fn func(board string, file assets.File)) (
	err error,
) {
	r, err := prepared[q].Query()
	if err != nil {
		return
	}
	defer r.Close()

	var (
		board string
		file  assets.File
	)
	for r.Next() {
		err = r.Scan(&board, &file.Data, &file.Mime)
		if err != nil {
			return
		}

		fn(board, file)
	}
	return r.Err()
}

func updateBanners(board string) (err error) {
	r, err := prepared["load_banners"].Query(board)
	if err != nil {
		return
	}
	defer r.Close()

	files := make([]assets.File, 0, common.MaxNumBanners)
	for r.Next() {
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
	}
	err = r.Err()
	if err != nil {
		return
	}

	assets.Banners.Set(board, files)
	return
}

// Set loading animation for specific board. Nil file.Data means the default
// animation should be used.
func SetLoadingAnimation(board string, file assets.File) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return
	}
	defer RollbackOnError(tx, &err)

	_, err = tx.Stmt(prepared["clear_loading"]).Exec(board)
	if err != nil {
		return
	}

	if file.Data != nil {
		_, err = tx.Stmt(prepared["set_loading"]).
			Exec(board, file.Data, file.Mime)
		if err != nil {
			return
		}
	}

	_, err = tx.Exec("select pg_notify('loading_animation_updated', $1)", board)
	if err != nil {
		return
	}

	err = tx.Commit()
	return
}

func loadLoadingAnimations() (err error) {
	err = loadAllAssets("load_all_loading", func(b string, f assets.File) {
		assets.Loading.Set(b, f)
	})
	if err != nil {
		return
	}

	return Listen("loading_animation_updated", updateLoadingAnimation)
}

func updateLoadingAnimation(board string) (err error) {
	var f assets.File
	err = prepared["load_loading"].QueryRow(board).Scan(&f.Data, &f.Mime)
	switch err {
	case nil, sql.ErrNoRows:
		assets.Loading.Set(board, f)
		err = nil
	}
	return
}
