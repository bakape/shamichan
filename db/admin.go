package db

import (
	"database/sql"
	"fmt"
	"strconv"

	"github.com/Masterminds/squirrel"
	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/imager/assets"
)

// Write moderation action to board-level and post-level logs
func logModeration(tx *sql.Tx, e auth.ModLogEntry) (err error) {
	_, err = sq.Insert("mod_log").
		Columns("type", "board", "post_id", "by", "length", "data").
		Values(e.Type, e.Board, e.ID, e.By, e.Length, e.Data).
		RunWith(tx).
		Exec()
	return
}

// Write redirects to db. /all/ seems like a good place for admin-only actions
func Redirect(id uint64, act common.ModerationAction, url string) (err error) {
	return InTransaction(false, func(tx *sql.Tx) (err error) {
		return logModeration(tx, auth.ModLogEntry{
			Board: "all",
			ID:    id,
			ModerationEntry: common.ModerationEntry{
				Type: act,
				By:   "admin",
				Data: url,
			},
		})
	})
}

// Clear post contents and remove any uploaded image from the server
func PurgePost(tx *sql.Tx, id uint64, by, reason string, by_ip bool) (
	err error,
) {
	type Post struct {
		ID    uint64
		Image struct {
			SHA1                sql.NullString
			FileType, ThumbType sql.NullInt64
		}
	}
	var posts []Post
	var board string
	var ip string

	err = sq.Select("board", "ip").
		From("posts").
		Where("id = ?", id).
		RunWith(tx).
		QueryRow().
		Scan(&board, &ip)
	if err != nil {
		return
	}

	getPosts := sq.Select("p.id", "i.SHA1", "i.file_type", "i.thumb_type").
		From("posts as p").
		LeftJoin("images as i on p.SHA1 = i.SHA1").
		RunWith(tx)

	if by_ip {
		getPosts = getPosts.Where("p.ip = ?", ip).
			Where("post_board(p.id) = ?", board)
	} else {
		getPosts = getPosts.Where("id = ?", id)
	}

	// Get all posts
	err = queryAll(
		getPosts,
		func(r *sql.Rows) (err error) {
			var post Post
			err = r.Scan(
				&post.ID,
				&post.Image.SHA1,
				&post.Image.FileType,
				&post.Image.ThumbType,
			)
			if err != nil {
				return
			}
			posts = append(posts, post)
			return
		},
	)
	if err != nil {
		return
	}

	for _, p := range posts {
		if p.Image.SHA1.Valid {
			img := p.Image
			_, err = sq.
				Delete("images").
				Where("sha1 = ?", img.SHA1.String).
				RunWith(tx).
				Exec()
			if err != nil {
				return
			}
			err = assets.Delete(
				img.SHA1.String,
				uint8(img.FileType.Int64),
				uint8(img.ThumbType.Int64),
			)
			if err != nil {
				return
			}
		}

		_, err = sq.
			Update("posts").
			Set("body", "").
			Where("id = ?", p.ID).
			RunWith(tx).
			Exec()
		if err != nil {
			return
		}

		err = logModeration(tx, auth.ModLogEntry{
			Board: board,
			ID:    p.ID,
			ModerationEntry: common.ModerationEntry{
				Type: common.PurgePost,
				By:   by,
				Data: reason,
			},
		})
		if err != nil {
			return
		}
	}

	return
}

// Apply post moderation, log and propagate to connected clients.
// query: optional query to execute on the post
func moderatePost(id uint64, entry common.ModerationEntry,
	query *squirrel.UpdateBuilder,
) (err error) {
	board, err := GetPostBoard(id)
	if err != nil {
		return
	}

	return InTransaction(false, func(tx *sql.Tx) (err error) {
		if query != nil {
			_, err = query.Where("id = ?", id).
				RunWith(tx).
				Exec()
			if err != nil {
				return
			}
		}
		return logModeration(tx, auth.ModLogEntry{
			ModerationEntry: entry,
			ID:              id,
			Board:           board,
		})
	})
}

// DeleteImage permanently deletes an image from a post
func DeleteImages(tx *sql.Tx, id uint64, by string, by_IP bool) (err error) {
	_, err = tx.Exec("select delete_images($1::bigint, $2::text, $3::boolean)",
		id, by, by_IP)

	castPermissionError(&err)
	return
}

// DeleteBoard deletes a board and all of its contained threads and posts
func DeleteBoard(board, by string) error {
	if board == "all" {
		return common.ErrInvalidInput("can not delete /all/")
	}
	return InTransaction(false, func(tx *sql.Tx) error {
		return deleteBoard(tx, board, by,
			fmt.Sprintf("board %s deleted by user", board))
	})
}

// ModSpoilerImage spoilers image as a moderator
func ModSpoilerImages(tx *sql.Tx, id uint64, by string, by_IP bool) (
	err error,
) {
	_, err = tx.Exec(
		"select spoiler_images($1::bigint, $2::text, $3::boolean)",
		id, by, by_IP,
	)

	castPermissionError(&err)
	return
}

// WriteStaff writes staff positions of a specific board. Old rows are
// overwritten.
func WriteStaff(tx *sql.Tx, board string,
	staff map[common.ModerationLevel][]string,
) (err error) {
	// Remove previous staff entries
	_, err = sq.Delete("staff").
		Where("board  = ?", board).
		RunWith(tx).
		Exec()
	if err != nil {
		return
	}

	// Write new ones
	q, err := tx.Prepare(`insert into staff (board, account, position)
		values($1, $2, $3)`)
	if err != nil {
		return
	}
	for pos, accounts := range staff {
		for _, a := range accounts {
			_, err = q.Exec(board, a, pos)
			if err != nil {
				return
			}
		}
	}

	return
}

// GetStaff retrieves all staff positions of a specific board
func GetStaff(board string,
) (staff map[common.ModerationLevel][]string, err error) {
	staff = make(map[common.ModerationLevel][]string, 3)
	err = queryAll(
		sq.Select("account", "position").
			From("staff").
			Where("board = ?", board),
		func(r *sql.Rows) (err error) {
			var (
				acc string
				pos common.ModerationLevel
			)
			err = r.Scan(&acc, &pos)
			if err != nil {
				return
			}
			staff[pos] = append(staff[pos], acc)
			return
		})
	return
}

// CanPerform returns, if the account can perform an action of ModerationLevel
// 'action' on the target board
func CanPerform(account, board string, action common.ModerationLevel) (
	can bool, err error,
) {
	switch {
	case account == "admin": // admin account can do anything
		return true, nil
	case action == common.Admin: // Only admin account can perform Admin actions
		return false, nil
	}

	pos, err := FindPosition(board, account)
	can = pos >= action
	if err == sql.ErrNoRows {
		err = nil
	}
	return
}

// GetSameIPPosts returns posts with the same IP and on the same board as the
// target post
func GetSameIPPosts(id uint64, board string, by string) (
	posts []common.StandalonePost, err error,
) {
	err = InTransaction(false, func(tx *sql.Tx) (err error) {
		// Get posts ids
		ids := make([]uint64, 0, 64)
		err = queryAll(
			sq.Select("id").
				From("posts").
				Where(`ip = (select ip from posts where id = ?) and board = ?`,
					id, board),
			func(r *sql.Rows) (err error) {
				var id uint64
				err = r.Scan(&id)
				if err != nil {
					return
				}
				ids = append(ids, id)
				return
			},
		)
		if err != nil {
			return
		}

		// Read the matched posts
		posts = make([]common.StandalonePost, 0, len(ids))
		var post common.StandalonePost

		for _, id := range ids {
			post, err = GetPost(id)

			switch err {
			case nil:
				posts = append(posts, post)
			case sql.ErrNoRows: // Deleted in race
				err = nil
			default:
				return
			}
		}
		return
	})

	err = moderatePost(id,
		common.ModerationEntry{
			Type: common.MeidoVision,
			By:   by,
		},
		nil)
	return
}

func castPermissionError(err *error) {
	if extractException(*err) == "access denied" {
		*err = common.ErrNoPermissions
	}
}

// DeletePost marks the target post as deleted
func DeletePosts(tx *sql.Tx, id uint64, by string, by_IP bool) (err error) {
	_, err = tx.Exec("select delete_posts($1::bigint, $2::text, $3::boolean)",
		id, by, by_IP)

	castPermissionError(&err)
	return
}

// SetThreadSticky sets the sticky field on a thread
func SetThreadSticky(id uint64, sticky bool) error {
	_, err := sq.Update("threads").
		Set("sticky", sticky).
		Where("id = ?", id).
		Exec()
	return err
}

// SetThreadLock sets the ability of users to post in a specific thread
func SetThreadLock(id uint64, locked bool, by string) error {
	q := sq.Update("threads").
		Set("locked", locked).
		Where("id = ?", id)
	return moderatePost(id,
		common.ModerationEntry{
			Type: common.LockThread,
			By:   by,
			Data: strconv.FormatBool(locked),
		},
		&q)
}

// GetModLog retrieves the moderation log for a specific board
func GetModLog(board string) (log []auth.ModLogEntry, err error) {
	log = make([]auth.ModLogEntry, 0, 64)
	e := auth.ModLogEntry{Board: board}
	err = queryAll(
		sq.
			Select(
				"ml.type",
				"ml.post_id",
				"ml.by",
				"ml.created",
				"ml.length",
				"ml.data",
				"coalesce(p.ip::text, '')",
			).
			From("mod_log as ml").
			LeftJoin("posts p on p.id = ml.post_id").
			Where("ml.board = ?", board).
			OrderBy("ml.created desc"),
		func(r *sql.Rows) (err error) {
			err = r.Scan(
				&e.Type,
				&e.ID,
				&e.By,
				&e.Created,
				&e.Length,
				&e.Data,
				&e.IP,
			)
			if err != nil {
				return
			}
			log = append(log, e)
			return
		},
	)
	return
}

// GetModLog retrieves the moderation log entry by ID
func GetModLogEntry(id uint64) (e auth.ModLogEntry, err error) {
	err = sq.
		Select("type", "board", "post_id", "by", "created", "length",
			"data").
		From("mod_log").
		Where("id = ?", id).
		QueryRow().
		Scan(&e.Type, &e.Board, &e.ID, &e.By, &e.Created, &e.Length,
			&e.Data)
	return
}
