package db

import (
	"io"
	"time"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/imager/assets"
	"github.com/bakape/meguca/util"
	"github.com/jackc/pgx"
)

var (
	// ErrInvalidToken occurs, when trying to retrieve an image with an
	// non-existent token. The token might have expired (60 to 119 seconds) or
	// the client could have provided an invalid token to begin with.
	ErrInvalidToken = common.ErrInvalidInput("invalid image token")
)

// Video structure
type Video struct {
	FileType uint8         `json:"file_type"`
	Duration time.Duration `json:"-"`
	SHA1     string        `json:"sha1"`
}

// WriteImage writes a processed image record to the DB. Only used in tests.
func WriteImage(i common.ImageCommon) error {
	return InTransaction(func(tx *pgx.Tx) error {
		return writeImageTx(tx, i)
	})
}

func writeImageTx(tx *pgx.Tx, i common.ImageCommon) (err error) {
	_, err = tx.Exec(
		"insert_image",
		i.Audio,
		i.Video,
		i.FileType,
		i.ThumbType,
		i.Dims,
		i.Length,
		i.Size,
		i.MD5,
		i.SHA1,
		i.Title,
		i.Artist,
	)
	return
}

// NewImageToken inserts a new image allocation token into the DB and returns
// it's ID
func NewImageToken(tx *pgx.Tx, SHA1 string) (token string, err error) {
	// Loop in case there is a primary key collision
	for {
		token, err = auth.RandomID(64)
		if err != nil {
			return
		}

		_, err = tx.Exec("insert_image_token", token, SHA1)
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

// ImageExists returns, if image exists
func ImageExists(tx *pgx.Tx, sha1 string) (exists bool, err error) {
	err = tx.QueryRow("image_exists", sha1).Scan(&exists)
	return
}

// AllocateImage allocates an image's file resources to their respective served
// directories and write its data to the database
func AllocateImage(
	tx *pgx.Tx,
	src,
	thumb io.ReadSeeker,
	img common.ImageCommon,
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
	err = db.QueryRow("has_image", id).Scan(&has)
	return
}

// InsertImage insert and image into and existing open post and return image
// JSON
func InsertImage(
	tx *pgx.Tx,
	postID uint64,
	token, name string,
	spoiler bool,
) (
	json []byte, err error,
) {
	err = tx.
		QueryRow(
			"insert_image_into_post",
			postID,
			token,
			name,
			spoiler,
		).
		Scan(&json)
	if extractException(err) == "invalid image token" {
		err = ErrInvalidToken
	}
	return
}

// GetImage retrieves a thumbnailed image record from the DB.
//
// Only used in tests.
func GetImage(sha1 string) (img common.ImageCommon, err error) {
	err = db.
		QueryRow(
			`select to_jsonb(i)
			from images i
			where sha1 = $1`,
			sha1,
		).
		Scan(&img)
	return
}

// SpoilerImage spoilers an already allocated image
func SpoilerImage(id, op uint64) error {
	_, err := db.Exec("spoiler_image", id)
	return err
}

// VideoPlaylist returns a video playlist for a board
func VideoPlaylist(board string) (videos []Video, err error) {
	videos = make([]Video, 0, 128)

	r, err := db.Query("get_video_playlist", board)
	if err != nil {
		return
	}
	defer r.Close()

	var (
		v   Video
		dur uint64
	)
	for r.Next() {
		err = r.Scan(&v.SHA1, &v.FileType, &dur)
		if err != nil {
			return
		}
		v.Duration = time.Duration(dur) * time.Second
		videos = append(videos, v)
	}
	err = r.Err()
	return
}

// Delete images not used in any posts
func deleteUnusedImages() (err error) {
	r, err := db.Query("delete_unused_images")
	if err != nil {
		return
	}
	defer r.Close()

	var (
		sha1                string
		fileType, thumbType uint8
	)
	for r.Next() {
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
