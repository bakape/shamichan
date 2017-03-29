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
		err = execPrepared("ban_post", post.id, post.op, msg)
		if err != nil {
			return
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
	return execPrepared("delete_post", id, msg)
}
