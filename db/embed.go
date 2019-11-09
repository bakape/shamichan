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
	err = db.
		QueryRow(
			`select title
			from bitchute_videos
			where id = $1`,
			id,
		).
		Scan(&title)
	return
}
