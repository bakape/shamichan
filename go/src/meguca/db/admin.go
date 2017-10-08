package db

import (
	"database/sql"
	"meguca/auth"
	"meguca/common"
	"time"
)

// Ban IPs from accessing a specific board. Need to target posts. Returns all
// banned IPs.
func Ban(board, reason, by string, expires time.Time, ids ...uint64) (
	ips map[string]uint64, err error,
) {
	type post struct {
		id, op uint64
	}

	// Retrieve matching posts
	ips = make(map[string]uint64, len(ids))
	posts := make([]post, 0, len(ids))
	for _, id := range ids {
		ip, err := GetIP(id)
		switch err {
		case nil:
		case sql.ErrNoRows:
			continue
		default:
			return nil, err
		}
		ips[ip] = id
		posts = append(posts, post{id: id})
	}

	// Retrieve their OPs
	for i, post := range posts {
		post.op, err = GetPostOP(post.id)
		if err != nil {
			return
		}
		posts[i] = post
	}

	// Write ban messages to posts
	for _, post := range posts {
		err = execPrepared("ban_post", post.id)
		if err != nil {
			return
		}
		if !IsTest {
			err = common.BanPost(post.id, post.op)
			if err != nil {
				return
			}
		}
	}

	// Write bans to the ban table
	for ip, id := range ips {
		err = execPrepared("write_ban", ip, board, id, reason, by, expires)
		if err != nil {
			return
		}
	}

	if len(ips) != 0 {
		_, err = db.Exec(`notify bans_updated`)
	}
	return
}

// Lift a ban from a specific post on a specific board
func Unban(board string, id uint64, by string) error {
	return execPrepared("unban", board, id, by)
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
	r, err := db.Query(`SELECT ip, board FROM bans`)
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

	return nil
}

// DeletePost marks the target post as deleted
func DeletePost(id uint64, by string) error {
	return moderatePost(id, by, "delete_post", common.DeletePost)
}

func moderatePost(
	id uint64,
	by, query string,
	propagate func(id, op uint64) error,
) (
	err error,
) {
	err = execPrepared(query, id, by)
	if err != nil {
		return
	}

	op, err := GetPostOP(id)
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
	return moderatePost(id, by, "delete_image", common.DeleteImage)
}

// Spoiler image as a moderator
func ModSpoilerImage(id uint64, by string) (err error) {
	return moderatePost(id, by, "mod_spoiler_image", common.SpoilerImage)
}

// WriteStaff writes staff positions of a specific board. Old rows are
// overwritten. tx must not be nil.
func WriteStaff(tx *sql.Tx, board string, staff map[string][]string) error {
	// Remove previous staff entries
	_, err := tx.Stmt(prepared["clear_staff"]).Exec(board)
	if err != nil {
		return err
	}

	// Write new ones
	q := tx.Stmt(prepared["write_staff"])
	for pos, accounts := range staff {
		for _, a := range accounts {
			_, err = q.Exec(board, a, pos)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// GetStaff retrieves all staff positions of a specific board
func GetStaff(board string) (staff map[string][]string, err error) {
	staff = make(map[string][]string, 3)
	r, err := prepared["get_staff"].Query(board)
	if err != nil {
		return
	}
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
	r, err := prepared["get_same_ip_posts"].Query(id, board)
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

// Set the sticky field on a thread
func SetThreadSticky(id uint64, sticky bool) error {
	return execPrepared("set_sticky", id, sticky)
}

// Set the ability of users to post in a specific thread
func SetThreadLock(id uint64, locked bool, by string) error {
	return execPrepared("set_locked", id, locked, by)
}

// Retrieve moderation log for a specific board
func GetModLog(board string) (log []auth.ModLogEntry, err error) {
	r, err := prepared["get_mod_log"].Query(board)
	if err != nil {
		return
	}
	defer r.Close()

	log = make([]auth.ModLogEntry, 0, 64)
	var entry auth.ModLogEntry
	for r.Next() {
		err = r.Scan(&entry.Type, &entry.ID, &entry.By, &entry.Created)
		if err != nil {
			return
		}
		log = append(log, entry)
	}
	err = r.Err()

	return
}
