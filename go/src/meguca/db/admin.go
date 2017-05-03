package db

import (
	"database/sql"
	"meguca/auth"
	"meguca/common"
	"time"
)

// ModerationLevel defines the level required to perform an action
type ModerationLevel int8

// All available moderation levels
const (
	NotStaff ModerationLevel = iota - 1
	Janitor
	Moderator
	BoardOwner
	Admin
)

// Ban IPs from accessing a specific board. Need to target posts. Returns all
// banned IPs.
func Ban(board, reason, by string, expires time.Time, ids ...uint64) (
	ips map[string]bool, err error,
) {
	type post struct {
		id, op uint64
	}

	// Retrieve matching posts
	ips = make(map[string]bool, len(ids))
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
		ips[ip] = true
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
		var msg []byte
		msg, err = common.EncodeMessage(common.MessageBanned, post.id)
		if err != nil {
			return
		}
		err = execPrepared("ban_post", post.id)
		if err != nil {
			return
		}
		if !IsTest {
			common.SendTo(post.op, msg)
		}
	}

	// Write bans to the ban table
	for ip := range ips {
		err = execPrepared("write_ban", ip, board, reason, by, expires)
		if err != nil {
			return
		}
	}

	if len(ips) != 0 {
		_, err = db.Exec(`notify bans_updated`)
	}
	return
}

func loadBans() error {
	if err := RefreshBanCache(); err != nil {
		return err
	}
	return listenFunc("bans_updated", func(_ string) error {
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
func DeletePost(board string, id uint64) (err error) {
	b, err := GetPostBoard(id)
	switch {
	case err == sql.ErrNoRows || b != board:
		return common.ErrInvalidPostID(id)
	case err != nil:
		return
	}

	msg, err := common.EncodeMessage(common.MessageDeletePost, id)
	if err != nil {
		return
	}

	err = execPrepared("delete_post", id)
	if err != nil {
		return
	}

	op, err := GetPostOP(id)
	if err != nil {
		return
	}
	if !IsTest {
		common.SendTo(op, msg)
	}

	return
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
func CanPerform(account, board string, action ModerationLevel) (
	can bool, err error,
) {
	switch {
	case account == "admin": // admin account can do anything
		return true, nil
	case action == Admin: // Only admin account can perform Admin actions
		return false, nil
	}

	r, err := prepared["get_positions"].Query(account, board)
	if err != nil {
		return
	}
	defer r.Close()

	// Read the highest position held
	pos := NotStaff
	for r.Next() {
		var s string
		err = r.Scan(&s)
		if err != nil {
			return
		}

		level := NotStaff
		switch s {
		case "owners":
			level = BoardOwner
		case "moderators":
			level = Moderator
		case "janitors":
			level = Janitor
		}
		if level > pos {
			pos = level
		}
	}
	err = r.Err()
	if err != nil {
		return
	}

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
