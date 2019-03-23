package db

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

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

// Clear post contents and remove any uploaded image from the server
func PurgePost(id uint64, by, reason string) (err error) {
	post, err := GetPost(id)
	if err != nil {
		return
	}
	return InTransaction(false, func(tx *sql.Tx) (err error) {
		if post.Image != nil {
			img := post.Image
			_, err = sq.
				Delete("images").
				Where("sha1 = ?", img.SHA1).
				RunWith(tx).
				Exec()
			if err != nil {
				return
			}
			err = assets.Delete(img.SHA1, img.FileType, img.ThumbType)
			if err != nil {
				return
			}
		}

		_, err = sq.
			Update("posts").
			Set("body", "").
			Where("id = ?", post.ID).
			RunWith(tx).
			Exec()
		if err != nil {
			return
		}

		return logModeration(tx, auth.ModLogEntry{
			Board: post.Board,
			ID:    post.ID,
			ModerationEntry: common.ModerationEntry{
				Type: common.PurgePost,
				By:   by,
				Data: reason,
			},
		})
	})
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
func DeleteImage(id uint64, by string) error {
	q := sq.Update("posts").Set("SHA1", nil)
	return moderatePost(id,
		common.ModerationEntry{
			Type: common.DeleteImage,
			By:   by,
		},
		&q)
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
func ModSpoilerImage(id uint64, by string) error {
	q := sq.Update("posts").Set("spoiler", true)
	return moderatePost(id,
		common.ModerationEntry{
			Type: common.SpoilerImage,
			By:   by,
		},
		&q)
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

// Delete posts of the same IP as target post on board and optionally keep
// deleting posts by this IP
func DeletePostsByIP(id uint64, account string, keepDeleting time.Duration,
	reason string,
) (err error) {
	seconds := 0
	if keepDeleting != 0 {
		seconds = int(keepDeleting / time.Second)
	}
	_, err = db.Exec(
		"select delete_posts_by_ip($1::bigint, $2::text, $3::bigint, $4::text)",
		id, account, seconds, reason)
	castPermissionError(&err)
	return
}

// Clear expired rules to delete posts on insertion
func clearExpiredContinuosDeletion() (err error) {
	_, err = sq.Delete("continuous_deletions").
		Where("till < ?", time.Now().UTC()).
		Exec()
	return
}

func castPermissionError(err *error) {
	if extractException(*err) == "access denied" {
		*err = common.ErrNoPermissions
	}
}

// DeletePost marks the target post as deleted
func DeletePost(id uint64, by string) (err error) {
	_, err = db.Exec("select delete_post($1::bigint, $2::text)",
		id, by)
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
		sq.Select("type", "post_id", "by", "created", "length", "data").
			From("mod_log").
			Where("board = ?", board).
			OrderBy("created desc"),
		func(r *sql.Rows) (err error) {
			err = r.Scan(&e.Type, &e.ID, &e.By, &e.Created, &e.Length,
				&e.Data)
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
