package db

import (
	"database/sql"
	"fmt"
	"meguca/auth"
	"meguca/common"
	"strconv"

	"github.com/Masterminds/squirrel"
)

// Write moderation action to board-level and post-level logs
func logModeration(tx *sql.Tx, op uint64, e auth.ModLogEntry) (err error) {
	err = withTransaction(tx,
		sq.Insert("mod_log").
			Columns("type", "board", "id", "by", "length", "data").
			Values(e.Type, e.Board, e.ID, e.By, e.Length, e.Data)).
		Exec()
	if err != nil {
		return
	}

	switch e.Type {
	case common.BanPost, common.DeletePost, common.DeleteImage,
		common.SpoilerImage, common.LockThread, common.MeidoVision:
		err = withTransaction(tx, sq.
			Insert("post_moderation").
			Columns("post_id", "type", "by", "length", "data").
			Values(e.ID, e.Type, e.By, e.Length, e.Data)).
			Exec()
		if err != nil {
			return
		}
		err = bumpThread(tx, op, false)
		if err != nil {
			return
		}
		if !IsTest {
			err = common.PropagateModeration(e.ID, op, e.ModerationEntry)
			if err != nil {
				return
			}
		}
	}
	return
}

// DeletePost marks the target post as deleted
func DeletePost(id uint64, by string) error {
	return moderatePost(id,
		common.ModerationEntry{
			Type: common.DeletePost,
			By:   by,
		},
		nil)
}

// Apply post moderation, log and propagate to connected clients.
// query: optional query to execute on the post
func moderatePost(id uint64, entry common.ModerationEntry,
	query *squirrel.UpdateBuilder,
) (err error) {
	board, op, err := GetPostParenthood(id)
	if err != nil {
		return
	}

	return InTransaction(false, func(tx *sql.Tx) (err error) {
		err = withTransaction(tx, sq.Update("posts").Set("moderated", true)).
			Exec()
		if err != nil {
			return
		}
		if query != nil {
			err = withTransaction(tx, query.Where("id = ?", id)).Exec()
			if err != nil {
				return
			}
		}
		return logModeration(tx, op, auth.ModLogEntry{
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
func WriteStaff(tx *sql.Tx, board string, staff map[string][]string) (
	err error,
) {
	// Remove previous staff entries
	err = withTransaction(tx, sq.Delete("staff").Where("board  = ?", board)).
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
func GetStaff(board string) (staff map[string][]string, err error) {
	staff = make(map[string][]string, 3)
	err = queryAll(
		sq.Select("account", "position").
			From("staff").
			Where("board = ?", board),
		func(r *sql.Rows) (err error) {
			var acc, pos string
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
func CanPerform(account, board string, action auth.ModerationLevel) (
	can bool, err error,
) {
	switch {
	case account == "admin": // admin account can do anything
		return true, nil
	case action == auth.Admin: // Only admin account can perform Admin actions
		return false, nil
	}

	pos, err := FindPosition(board, account)
	can = pos >= action
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

// SetThreadSticky sets the sticky field on a thread
func SetThreadSticky(id uint64, sticky bool) error {
	return InTransaction(false, func(tx *sql.Tx) (err error) {
		err = withTransaction(tx,
			sq.Update("threads").
				Set("sticky", sticky).
				Where("id = ?", id)).
			Exec()
		if err != nil {
			return
		}
		return bumpThread(tx, id, false)
	})
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
		sq.Select("type", "id", "by", "created", "length", "data").
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
