package db

import (
	"context"
	"errors"
	"io"

	"github.com/bakape/meguca/imager/assets"
	"github.com/bakape/meguca/imager/common"
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
	_, err = tx.Exec(ctx, q, args...)
	if err != nil {
		return
	}
	return assets.Write(img.SHA1, img.FileType, img.ThumbType, src, thumb)
}

// Insert and image into and existing open post. Returns the post's thread.
//
// Returns pgx.ErrNoRows, if no open post for the target pubKey was found.
func InsertImage(
	ctx context.Context,
	tx pgx.Tx,
	post, pubKey uint64,
	img common.SHA1Hash,
	name string,
	spoilered bool,
) (
	thread uint64,
	err error,
) {
	err = tx.
		QueryRow(
			ctx,
			`update posts
			set image = $1,
				image_name = $2,
				image_spoilered = $3
			where open and public_key = $4 and id = $5 and image is null
			returning thread`,
			img,
			name,
			spoilered,
			pubKey,
			post,
		).
		Scan(&thread)
	return
}

// Return, if pubKey has any post that an image can be inserted into
func ScheduleImageProcessing(
	ctx context.Context,
	post uint64,
	pubKey uint64,
	src []byte,
) (err error) {
	return InTransaction(ctx, func(tx pgx.Tx) (err error) {
		var isOpen, noImage bool
		err = tx.
			QueryRow(
				ctx,
				`select open, image is null
				from posts
				where id = $1 and public_key = $2`,
				post, pubKey,
			).
			Scan(&isOpen, noImage)
		switch err {
		case nil:
		case pgx.ErrNoRows:
			return errors.New("post not found")
		default:
			return
		}
		if !isOpen {
			return errors.New("post already closed")
		}
		if !noImage {
			return errors.New("post already has an image")
		}
		_, err = tx.Exec(
			ctx,
			`insert into pending_images (post, source) values ($1, $2)`,
			post, src,
		)
		return
	})
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

			&img.Title,
			&img.Artist,
		)
	if err != nil {
		return
	}
	img.SHA1 = id
	return
}
