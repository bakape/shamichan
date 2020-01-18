package db

import (
	"context"
	"io"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/imager/assets"
	"github.com/bakape/pg_util"
	"github.com/jackc/pgx/v4"
)

// AllocateImage allocates an image's file resources to their respective served
// directories and write its data to the database
func AllocateImage(
	ctx context.Context,
	tx pgx.Tx,
	img common.ImageCommon,
	src, thumb io.ReadSeeker,
) (
	err error,
) {
	q, args := pg_util.BuildInsert(pg_util.InsertOpts{
		Table: "images",
		Data:  img,
	})
	// for i := range args {
	// 	_, ok := args[i].(uint16)
	// 	if ok {
	// 		args[i] = fmt.Sprint(args[i])
	// 	}
	// }
	_, err = tx.Exec(ctx, q, args...)
	if err != nil {
		return
	}
	return assets.Write(img.SHA1, img.FileType, img.ThumbType, src, thumb)
}

// Insert and image into and existing open post.
//
// Returns pgx.ErrNoRows, if no open post for the target user was found.
func InsertImage(
	ctx context.Context,
	tx pgx.Tx,
	user auth.AuthKey,
	img common.SHA1Hash,
	name string,
	spoilered bool,
) (err error) {
	res, err := tx.Exec(
		ctx,
		`update posts
		set image = $1,
			image_name = $2,
			image_spoilered = $3
		where open and auth_key = $4`,
		img,
		name,
		spoilered,
		user,
	)
	if err != nil {
		return
	}
	if res.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return
}

// Retrieves a thumbnailed image record from the DB.
// Protects it from possible concurrent deletes until the transaction closes.
func GetImage(ctx context.Context, tx pgx.Tx, id common.SHA1Hash) (
	img common.ImageCommon,
	err error,
) {
	err = db.
		QueryRow(
			context.Background(),
			`select
				md5,

				audio,
				video,

				file_type,
				thumb_type,

				width,
				height,
				thumb_width,
				thumb_height,

				size,
				duration,

				title,
				artist
			from images
			where sha1 = $1
			for update`,
			id,
		).
		Scan(
			&img.MD5,

			&img.Audio,
			&img.Video,

			&img.FileType,
			&img.ThumbType,

			&img.Width,
			&img.Height,
			&img.ThumbWidth,
			&img.ThumbHeight,

			&img.Size,
			&img.Duration,

			&img.Artist,
			&img.Title,
		)
	if err != nil {
		return
	}
	img.SHA1 = id
	return
}

// SpoilerImage spoilers an already allocated image
func SpoilerImage(ctx context.Context, id uint64) error {
	_, err := db.Exec(
		ctx,
		`update posts
		set spoiler = true
		where id = $1`,
		id,
	)
	return err
}

// Delete images not used in any posts
func deleteUnusedImages() (err error) {
	r, err := db.Query(
		context.Background(),
		`delete from images as i
		where not exists (
			select
			from posts p
			where p.image = i.sha1
		)
		returning i.sha1, i.file_type, i.thumb_type`,
	)
	if err != nil {
		return
	}
	var (
		sha1                common.SHA1Hash
		fileType, thumbType common.FileType
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
