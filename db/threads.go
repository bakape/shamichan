package db

import (
	"context"

	"github.com/bakape/pg_util"
	"github.com/jackc/pgx/v4"
)

type ThreadInsertParams struct {
	Subject string

	// Must include between 1 and 3 tags
	Tags []string

	PostInsertParamsCommon `db:"-"`
}

// Insert thread and empty post into DB and return the post ID.
//
// authKey is optional, in case this is a migration of a legacy post
func InsertThread(ctx context.Context, p ThreadInsertParams) (
	id uint64, err error,
) {
	err = InTransaction(nil, func(tx pgx.Tx) (err error) {
		q, args := pg_util.BuildInsert(pg_util.InsertOpts{
			Table:  "threads",
			Data:   p,
			Suffix: "returning id",
		})
		defer pg_util.ResuseArgs(args)
		err = tx.QueryRow(ctx, q, args...).Scan(&id)
		if err != nil {
			return
		}

		_, err = InsertPost(ctx, tx, OPInsertparams{
			ID: id,
			ReplyInsertParams: ReplyInsertParams{
				Thread:                 id,
				PostInsertParamsCommon: p.PostInsertParamsCommon,
			},
		})
		return
	})
	return
}

// Check, if thread exists in the database
func ThreadExists(ctx context.Context, id uint64) (exists bool, err error) {
	err = db.
		QueryRow(
			ctx,
			`select exists (
				select
				from threads
				where id = $1
			)`,
			id,
		).
		Scan(&exists)
	return
}

// Read feed data for initializing Pulsar as JSON
func GetFeedData() (buf []byte, err error) {
	err = db.
		QueryRow(
			context.Background(),
			`select jsonb_object_agg(thread, val)
			from (
				select thread, merge_jsonb_obj(val) val
				from (
					select
						thread,
						jsonb_build_object(
							'recent_posts', jsonb_object_agg(id, time)
						) val
					from posts
					where created_on > now() - interval '16 minutes'
					group by thread

					union all

					select
						thread,
						jsonb_build_object(
							'open_posts', jsonb_object_agg(
								id,
								jsonb_build_object(
									'has_image', image is not null,
									'image_spoilered', image_spoilered,
									'created_on', extract(
										epoch from created_on
									),
									'thread', thread,
									'body', body
								)
							)
						) val
					from posts
					where open
					group by thread
				) as d
				group by thread
			) as d`,
		).
		Scan(&buf)
	return
}
