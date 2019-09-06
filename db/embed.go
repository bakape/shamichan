package db

import (
	"fmt"
	"database/sql"

	"github.com/bakape/meguca/common"
)

// genericExists checks if db table already has a row for ID.
func genericExists(tx *sql.Tx, id string, table string) (e bool, err error) {
	err = sq.Select("count(1)").
		From(table).
		Where("id = ?", id).
		RunWith(tx).
		QueryRow().
		Scan(&e)
	return
}

// WriteYouTubeInfo creates a new youtube info row.
func WriteYouTubeInfo(tx *sql.Tx, id string, title string, thumb string, video string, videoHigh string) error {
	exists, err := genericExists(tx, id, "youtube_videos")

	if err != nil {
		return err
	}

	if exists {
		return errExists("YouTube", id, "WriteYouTubeInfo")
	}

	_, err = sq.Insert("youtube_videos").
		Columns("id", "title", "thumb", "video", "videoHigh").
		Values(id, title, thumb, video, videoHigh).
		RunWith(tx).
		Exec()
	return err
}

// GetYouTubeInfo retrieves the youtube video info by ID
func GetYouTubeInfo(tx *sql.Tx, id string) (title string, thumb string, video string, videoHigh string, err error) {
	err = sq.Select("title", "thumb", "video", "videoHigh").
		From("youtube_videos").
		Where("id = ?", id).
		RunWith(tx).
		QueryRow().
		Scan(&title, &thumb, &video, &videoHigh)
	return
}

// WriteBitChuteTitle creates a new bitchute title row.
func WriteBitChuteTitle(tx *sql.Tx, id string, title string) error {
	exists, err := genericExists(tx, id, "bitchute_videos")

	if err != nil {
		return err
	}

	if exists {
		return errExists("BitChute", id, "WriteBitChuteTitle")
	}

	_, err = sq.Insert("bitchute_videos").
		Columns("id", "title").
		Values(id, title).
		RunWith(tx).
		Exec()
	return err
}

// GetBitChuteTitle retrieves the bitchute video title by ID
func GetBitChuteTitle(tx *sql.Tx, id string) (title string, err error) {
	err = sq.Select("title").
		From("bitchute_videos").
		Where("id = ?", id).
		RunWith(tx).
		QueryRow().
		Scan(&title)
	return
}

func errExists(service string, id string, method string) error {
	return common.StatusError{fmt.Errorf("%s [%s]: Invalid %s method use, already exists in database", service, id, method), 500}
}
