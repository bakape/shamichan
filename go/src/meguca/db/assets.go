package db

import (
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
	r, err := prepared["load_all_banners"].Query()
	if err != nil {
		return
	}
	defer r.Close()

	// Load all banners and group by board
	byBoard := make(map[string][]assets.File, 64)
	for r.Next() {
		var (
			board, mime string
			data        []byte
		)
		err = r.Scan(&board, &data, &mime)
		if err != nil {
			return
		}

		files := byBoard[board]
		if files == nil {
			files = make([]assets.File, 0, common.MaxNumBanners)
		}
		byBoard[board] = append(files, assets.File{
			Data: data,
			Mime: mime,
		})
	}
	err = r.Err()
	if err != nil {
		return
	}

	for board, files := range byBoard {
		assets.Banners.Set(board, files)
	}

	return listenFunc("banners_updated", updateBanners)
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
