package db

import (
	"database/sql"
	"fmt"
	"meguca/auth"
	"meguca/common"
	"time"

	"github.com/Masterminds/squirrel"
)

// Write moderation action to log
func LogModeration(e auth.ModLogEntry) error {
	_, err := sq.Insert("mod_log").
		Columns("type", "board", "id", "by", "length", "reason").
		Values(int(e.Type), e.Board, e.ID, e.By, e.Length, e.Reason).
		Exec()

	return err
}

// Write moderation action to log atomically
func logModerationA(tx *sql.Tx, e auth.ModLogEntry) error {
	return withTransaction(tx,
		sq.Insert("mod_log").
			Columns("type", "board", "id", "by", "length", "reason").
			Values(int(e.Type), e.Board, e.ID, e.By, e.Length, e.Reason),
	).Exec()
}

func writeBan(
	tx *sql.Tx,
	ip, board, reason, by string,
	postID uint64,
	expires time.Time,
	log bool,
) (
	err error,
) {
	err = withTransaction(tx,
		sq.Insert("bans").
			Columns("ip", "board", "forPost", "reason", "by", "expires").
			Values(ip, board, postID, reason, by, expires.UTC()).
			Suffix("on conflict do nothing"),
	).
		Exec()
	if err != nil || !log {
		return
	}
	return logModerationA(tx, auth.ModLogEntry{
		Type:   auth.BanPost,
		Board:  board,
		ID:     postID,
		By:     by,
		Length: uint64(expires.Sub(time.Now()).Seconds()),
		Reason: reason,
	})
}

// Propagate ban updates through DB and disconnect all banned IPs
func propagateBans(board string, ips ...string) error {
	if len(ips) != 0 {
		if _, err := db.Exec(`notify bans_updated`); err != nil {
			return err
		}
	}
	if !IsTest {
		for _, ip := range ips {
			auth.DisconnectBannedIP(ip, board)
		}
	}
	return nil
}

// Automatically ban an IP
func SystemBan(ip, reason string, expires time.Time) (err error) {
	err = InTransaction(func(tx *sql.Tx) error {
		return writeBan(tx, ip, "all", reason, "system", 0, expires, true)
	})
	if err != nil {
		return
	}
	err = propagateBans("all", ip)
	return
}

// Ban IPs from accessing a specific board. Need to target posts. Returns all
// banned IPs.
func Ban(board, reason, by string, expires time.Time, log bool, ids ...uint64) (
	err error,
) {
	type post struct {
		id, op uint64
		ip     string
	}

	// Retrieve matching posts
	var (
		ips   = make(map[string]bool, len(ids))
		posts = make([]post, 0, len(ids))
		ip    string
	)
	for _, id := range ids {
		ip, err = GetIP(id)
		switch err {
		case nil:
		case sql.ErrNoRows:
			err = nil
			continue
		default:
			return
		}
		ips[ip] = true
		posts = append(posts, post{
			id: id,
			ip: ip,
		})
	}

	// Retrieve their OPs
	for i := range posts {
		posts[i].op, err = GetPostOP(posts[i].id)
		if err != nil {
			return
		}
	}

	// Write ban messages to posts and ban table
	err = InTransaction(func(tx *sql.Tx) (err error) {
		for _, post := range posts {
			err = withTransaction(tx,
				sq.Update("posts").
					Set("banned", true).
					Where("id = ?", post.id),
			).
				Exec()
			if err != nil {
				return
			}
			err = bumpThread(tx, post.op, false)
			if err != nil {
				return
			}
			err = writeBan(tx, post.ip, board, reason, by, post.id, expires,
				log)
			if err != nil {
				return
			}
		}
		return
	})
	if err != nil {
		return
	}

	if !IsTest {
		for _, post := range posts {
			err = common.BanPost(post.id, post.op)
			if err != nil {
				return
			}
		}
	}

	ipArr := make([]string, 0, len(ips))
	for ip, _ := range ips {
		ipArr = append(ipArr, ip)
	}
	return propagateBans(board, ipArr...)
}

// Lift a ban from a specific post on a specific board
func Unban(board string, id uint64, by string) error {
	return InTransaction(func(tx *sql.Tx) (err error) {
		err = withTransaction(tx,
			sq.Delete("bans").
				Where("board = ? and forPost = ?", board, id),
		).
			Exec()
		if err != nil {
			return
		}
		err = logModerationA(tx, auth.ModLogEntry{
			Type:  auth.UnbanPost,
			Board: board,
			ID:    id,
			By:    by,
		})
		if err != nil {
			return
		}
		_, err = tx.Exec("notify bans_updated")
		return
	})
}

func loadBans() error {
	if err := RefreshBanCache(); err != nil {
		return err
	}
	return Listen("bans_updated", func(_ string) error {
		return RefreshBanCache()
	})
}

// RefreshBanCache loads up to date bans from the database and caches them in
// memory
func RefreshBanCache() (err error) {
	r, err := sq.Select("ip", "board").From("bans").Query()
	if err != nil {
		return
	}
	defer r.Close()

	bans := make([]auth.Ban, 0, 16)
	for r.Next() {
		var b auth.Ban
		err = r.Scan(&b.IP, &b.Board)
		if err != nil {
			return
		}
		bans = append(bans, b)
	}
	err = r.Err()
	if err != nil {
		return
	}
	auth.SetBans(bans...)

	return
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

	err = InTransaction(func(tx *sql.Tx) (err error) {
		err = withTransaction(tx, query.Where("id = ?", id)).Exec()
		if err != nil {
			return
		}
		err = logModerationA(tx, auth.ModLogEntry{
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
		err = propagate(id, op)
	}
	return
}

// Permanently delete an image from a post
func DeleteImage(id uint64, by string) error {
	q := sq.Update("posts").Set("SHA1", nil)
	return moderatePost(id, auth.DeleteImage, by, q, common.DeleteImage)
}

// DeleteBoard deletes a board and all of its contained threads and posts
func DeleteBoard(board, by string) error {
	if board == "all" {
		return common.ErrInvalidInput("can not delete /all/")
	}
	return InTransaction(func(tx *sql.Tx) error {
		return deleteBoard(tx, board, by,
			fmt.Sprintf("board %s deleted by user", board))
	})
}

// Spoiler image as a moderator
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
	r, err := sq.Select("account", "position").
		From("staff").
		Where("board = ?", board).
		Query()
	if err != nil {
		return
	}
	defer r.Close()

	for r.Next() {
		var acc, pos string
		err = r.Scan(&acc, &pos)
		if err != nil {
			return
		}
		staff[pos] = append(staff[pos], acc)
	}

	err = r.Err()
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
func GetSameIPPosts(id uint64, board string) (
	posts []common.StandalonePost, err error,
) {
	// Get posts ids
	r, err := sq.Select("id").
		From("posts").
		Where(`ip = (select ip from posts where id = ?) and board = ?`,
			id, board).
		Query()
	if err != nil {
		return
	}
	defer r.Close()
	var ids = make([]uint64, 0, 64)
	for r.Next() {
		var id uint64
		err = r.Scan(&id)
		if err != nil {
			return
		}
		ids = append(ids, id)
	}
	err = r.Err()
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
}

func setThreadBool(id uint64, key string, val bool) error {
	return InTransaction(func(tx *sql.Tx) (err error) {
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

// Set the sticky field on a thread
func SetThreadSticky(id uint64, sticky bool) error {
	return setThreadBool(id, "sticky", sticky)
}

// Set the ability of users to post in a specific thread
func SetThreadLock(id uint64, locked bool, by string) error {
	return setThreadBool(id, "locked", locked)
}

// Retrieve moderation log for a specific board
func GetModLog(board string) (log []auth.ModLogEntry, err error) {
	r, err := sq.Select("type", "id", "by", "created", "length", "reason").
		From("mod_log").
		Where("board = ?", board).
		OrderBy("created desc").
		Query()
	if err != nil {
		return
	}
	defer r.Close()

	log = make([]auth.ModLogEntry, 0, 64)
	e := auth.ModLogEntry{Board: board}
	for r.Next() {
		err = r.Scan(&e.Type, &e.ID, &e.By, &e.Created, &e.Length, &e.Reason)
		if err != nil {
			return
		}
		log = append(log, e)
	}
	err = r.Err()
	return
}
