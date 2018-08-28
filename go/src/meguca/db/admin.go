package db

import (
	"fmt"
	"database/sql"

	"meguca/auth"
	"meguca/common"
	"meguca/templates"

	"github.com/Masterminds/squirrel"
)

// Export without circular dependency
func init() {
	templates.GetPostModLog = GetPostModLog
}

// Write moderation action to log
func logModeration(tx *sql.Tx, e auth.ModLogEntry) error {
	return withTransaction(tx,
		sq.Insert("mod_log").
			Columns("type", "board", "id", "by", "length", "reason").
			Values(int(e.Type), e.Board, e.ID, e.By, e.Length, e.Reason),
	).Exec()
}

// DeletePost marks the target post as deleted
func DeletePost(id uint64, by string) error {
	var del sql.NullBool
	err := sq.Select("deleted").From("posts").Where("id = ?", id).Scan(&del)
	if err != nil {
		return err
	}

	if !del.Bool {
		return moderatePost(id, auth.DeletePost, by,
			sq.Update("posts").Set("deleted", true), common.DeletePost)
	}

	return nil
}

func moderatePost(
	id uint64, typ auth.ModerationAction, by string,
	query squirrel.UpdateBuilder,
	propagate func(id, op uint64) error,
) (
	err error,
) {
	board, op, err := GetPostParenthood(id)
	if err != nil {
		return
	}

	err = InTransaction(false, func(tx *sql.Tx) (err error) {
		err = withTransaction(tx, query.Where("id = ?", id)).Exec()
		if err != nil {
			return
		}
		err = logModeration(tx, auth.ModLogEntry{
			Type:  typ,
			Board: board,
			ID:    id,
			By:    by,
		})
		if err != nil {
			return
		}
		return bumpThread(tx, op, false)
	})
	if err != nil {
		return
	}

	if !IsTest {
		switch typ {
		case auth.BanPost, auth.DeletePost, auth.MeidoVision:
			err = auth.ModLogPost(id, op, GetPostModLog(id))

			if err != nil {
				return
			}
		}

		err = propagate(id, op)
	}
	return
}

// DeleteImage permanently deletes an image from a post
func DeleteImage(id uint64, by string) error {
	q := sq.Update("posts").Set("SHA1", nil)
	return moderatePost(id, auth.DeleteImage, by, q, common.DeleteImage)
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
	return moderatePost(id, auth.SpoilerImage, by, q, common.SpoilerImage)
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

		return moderatePost(id, auth.MeidoVision, by,
			sq.Update("posts").Set("meidoVision", true), common.MeidoVisionPost)
	})

	return
}

func setThreadBool(id uint64, key string, val bool) error {
	return InTransaction(false, func(tx *sql.Tx) (err error) {
		err = withTransaction(tx,
			sq.Update("threads").
				Set(key, val).
				Where("id = ?", id),
		).
			Exec()
		if err != nil {
			return
		}
		return bumpThread(tx, id, false)
	})
}

// SetThreadSticky sets the sticky field on a thread
func SetThreadSticky(id uint64, sticky bool) error {
	return setThreadBool(id, "sticky", sticky)
}

// SetThreadLock sets the ability of users to post in a specific thread
func SetThreadLock(id uint64, locked bool, by string) error {
	return setThreadBool(id, "locked", locked)
}

// GetModLog retrieves the moderation log for a specific board
func GetModLog(board string) (log []auth.ModLogEntry, err error) {
	log = make([]auth.ModLogEntry, 0, 64)
	e := auth.ModLogEntry{Board: board}
	err = queryAll(
		sq.Select("type", "id", "by", "created", "length", "reason").
			From("mod_log").
			Where("board = ?", board).
			OrderBy("created desc"),
		func(r *sql.Rows) (err error) {
			err = r.Scan(&e.Type, &e.ID, &e.By, &e.Created, &e.Length,
				&e.Reason)
			if err != nil {
				return
			}
			log = append(log, e)
			return
		},
	)
	return
}

// GetPostModLog retrieves a post's (3) relevant mod-log entries
func GetPostModLog(id uint64) []auth.ModLogEntry {
	log := make([]auth.ModLogEntry, 3, 3)

	for i, val := range [3]auth.ModerationAction {
		auth.BanPost,
		auth.DeletePost,
		auth.MeidoVision,
	} {
		sq.Select("type", "id", "length", "created", "board", "by", "reason").
			From("mod_log").
			Where("type = ?", val).
			Where("id = ?", id).
			Scan(
				&log[i].Type,
				&log[i].ID,
				&log[i].Length,
				&log[i].Created,
				&log[i].Board,
				&log[i].By,
				&log[i].Reason,
			)
	}

	return log
}
