package db

// WriteBitChuteTitle creates a new bitchute title row.
func WriteBitChuteTitle(id string, title string) error {
	_, err := db.Exec(
		`insert into bitchute_videos (id, title)
		values ($1, $2)
		on conflict (id) do nothing`,
		id, title,
	)
	return err
}

// GetBitChuteTitle retrieves the bitchute video title by ID
func GetBitChuteTitle(id string) (title string, err error) {
	err = sq.Select("title").
		From("bitchute_videos").
		Where("id = ?", id).
		QueryRow().
		Scan(&title)
	return
}
