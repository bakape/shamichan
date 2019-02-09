package db

import (
	"database/sql"
	"errors"
	"io"
	"meguca/auth"
	"meguca/common"
	"meguca/imager/assets"
	"meguca/util"
	"time"

	"github.com/lib/pq"
)

const (
	// Time it takes for an image allocation token to expire
	tokenTimeout = time.Minute
)

var (
	// ErrInvalidToken occurs, when trying to retrieve an image with an
	// non-existent token. The token might have expired (60 to 119 seconds) or
	// the client could have provided an invalid token to begin with.
	ErrInvalidToken = errors.New("invalid image token")
)

// Video structure
type Video struct {
	FileType uint8         `json:"fileType"`
	Duration time.Duration `json:"-"`
	SHA1     string        `json:"sha1"`
}

// WriteImage writes a processed image record to the DB. Only used in tests.
func WriteImage(i common.ImageCommon) error {
	return InTransaction(false, func(tx *sql.Tx) error {
		return writeImageTx(tx, i)
	})
}

func writeImageTx(tx *sql.Tx, i common.ImageCommon) (err error) {
	_, err = sq.
		Insert("images").
		Columns(
			"audio", "video", "fileType", "thumbType", "dims", "length",
			"size", "MD5", "SHA1", "Title", "Artist",
		).
		Values(
			i.Audio, i.Video, int(i.FileType), int(i.ThumbType),
			pq.GenericArray{A: i.Dims}, i.Length, i.Size, i.MD5, i.SHA1,
			i.Title, i.Artist,
		).
		RunWith(tx).
		Exec()
	return
}

// NewImageToken inserts a new image allocation token into the DB and returns
// it's ID in a transacion
func NewImageToken(tx *sql.Tx, SHA1 string) (token string, err error) {
	expires := time.Now().Add(tokenTimeout).UTC()

	// Loop in case there is a primary key collision
	for {
		token, err = auth.RandomID(64)
		if err != nil {
			return
		}

		_, err = sq.
			Insert("image_tokens").
			Columns("token", "SHA1", "expires").
			Values(token, SHA1, expires).
			RunWith(tx).
			Exec()
		switch {
		case err == nil:
			return
		case IsConflictError(err):
			continue
		default:
			return
		}
	}
}

// AllocateImage allocates an image's file resources to their respective served
// directories and write its data to the database
func AllocateImage(tx *sql.Tx, src, thumb io.ReadSeeker, img common.ImageCommon,
) (
	err error,
) {
	err = writeImageTx(tx, img)
	if err != nil {
		return err
	}

	err = assets.Write(img.SHA1, img.FileType, img.ThumbType, src, thumb)
	if err != nil {
		return cleanUpFailedAllocation(img, err)
	}
	return nil
}

// Delete any dangling image files in case of a failed image allocation
func cleanUpFailedAllocation(img common.ImageCommon, err error) error {
	delErr := assets.Delete(img.SHA1, img.FileType, img.ThumbType)
	if delErr != nil {
		err = util.WrapError(err.Error(), delErr)
	}
	return err
}

// HasImage returns, if the post has an image allocated. Only used in tests.
func HasImage(id uint64) (has bool, err error) {
	err = sq.Select("true").
		From("posts").
		Where("id = ? and SHA1 IS NOT NULL", id).
		QueryRow().
		Scan(&has)
	if err == sql.ErrNoRows {
		err = nil
	}
	return
}

// InsertImage insert and image into and existing open post and return image
// JSON
func InsertImage(postID uint64, token, name string, spoiler bool) (
	json []byte, err error,
) {
	err = db.QueryRow(
		`select insert_image($1::bigint,
			$2::char(86),
			$3::varchar(200),
			$4::bool)`,
		postID, token, name, spoiler).
		Scan(&json)
	return
}

// SpoilerImage spoilers an already allocated image
func SpoilerImage(id, op uint64) error {
	_, err := sq.Update("posts").
		Set("spoiler", true).
		Where("id = ?", id).
		Exec()
	return err
}

// VideoPlaylist returns a video playlist for a board
func VideoPlaylist(board string) (videos []Video, err error) {
	videos = make([]Video, 0, 128)
	var (
		v   Video
		dur uint64
	)
	err = queryAll(
		sq.Select("i.SHA1", "i.fileType", "i.length").
			From("images as i").
			Where(`
				exists(select 1
					from posts as p
					where p.sha1 = i.sha1 and p.board = ?)
				and filetype in (?, ?)
				and audio = true
				and video = true
				and length between 10 and 600`,
				board,
				int(common.WEBM),
				int(common.MP4),
			).
			OrderBy("random()"),
		func(r *sql.Rows) (err error) {
			err = r.Scan(&v.SHA1, &v.FileType, &dur)
			if err != nil {
				return
			}
			v.Duration = time.Duration(dur) * time.Second
			videos = append(videos, v)
			return
		},
	)
	return
}

// ImageExists returns, if image exists
func ImageExists(sha1 string) (exists bool, err error) {
	err = sq.Select("1").
		From("images").
		Where("sha1 = ?", sha1).
		Scan(&exists)
	if err == sql.ErrNoRows {
		err = nil
	}
	return
}

// Delete images not used in any posts
func deleteUnusedImages() (err error) {
	r, err := db.Query(`
		delete from images
		where (
			(select count(*) from posts where SHA1 = images.SHA1)
			+ (select count(*) from image_tokens where SHA1 = images.SHA1)
		) = 0
		returning SHA1, fileType, thumbType`)
	if err != nil {
		return
	}
	defer r.Close()

	for r.Next() {
		var (
			sha1                string
			fileType, thumbType uint8
		)
		err = r.Scan(&sha1, &fileType, &thumbType)
		if err != nil {
			return
		}
		err = assets.Delete(sha1, fileType, thumbType)
		if err != nil {
			return
		}
	}

	return r.Err()
}
