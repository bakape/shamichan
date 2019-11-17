package db

// TODO: Moderation
// TODO: Thread-level bans. thread=0 => global ban

// // Write moderation action to board-level and post-level logs
// func logModeration(tx *pgx.Tx, e auth.ModLogEntry) (err error) {
// 	_, err = tx.Exec(
// 		`insert into mod_log (type, board, post_id, by, length, data)
// 		values($1, $2, $3, $4, $5, $6)`,
// 		e.Type, e.Board, e.ID, e.By, e.Length, e.Data,
// 	)
// 	return
// }

// // Clear post contents and remove any uploaded image from the server
// func PurgePost(id uint64, by, reason string) (err error) {
// 	return InTransaction(func(tx *pgx.Tx) (err error) {
// 		var (
// 			board               string
// 			hash                *string
// 			fileType, thumbType *uint8
// 		)
// 		err = tx.
// 			QueryRow(
// 				`select p.board, i.sha1, i.file_type, i.thumb_type
// 				from posts p
// 				left join images i on p.sha1 = i.sha1
// 				where p.id = $1`,
// 				id,
// 			).
// 			Scan(&board, &hash, &fileType, &thumbType)
// 		if err != nil {
// 			return
// 		}

// 		if hash != nil {
// 			_, err = tx.Exec("delete from images where sha1 = $1", *hash)
// 			if err != nil {
// 				return
// 			}

// 			err = assets.Delete(
// 				*hash,
// 				*fileType,
// 				*thumbType,
// 			)
// 			if err != nil {
// 				return
// 			}
// 		}

// 		_, err = tx.Exec(
// 			`update posts
// 			set body = ''
// 			where id = $1`,
// 			id,
// 		)
// 		if err != nil {
// 			return
// 		}

// 		return logModeration(tx, auth.ModLogEntry{
// 			Board: board,
// 			ID:    id,
// 			ModerationEntry: common.ModerationEntry{
// 				Type: common.PurgePost,
// 				By:   by,
// 				Data: reason,
// 			},
// 		})
// 	})
// }

// // DeleteImage permanently deletes an image from a post
// func DeleteImages(ids []uint64, by string) (err error) {
// 	_, err = db.Exec(
// 		"select delete_images($1::bigint[], $2::text)",
// 		encodeUint64Array(ids),
// 		by,
// 	)
// 	castPermissionError(&err)
// 	return
// }

// // ModSpoilerImage spoilers image as a moderator
// func ModSpoilerImages(ids []uint64, by string) (err error) {
// 	_, err = db.Exec(
// 		"select spoiler_images($1::bigint[], $2::text)",
// 		encodeUint64Array(ids),
// 		by,
// 	)
// 	castPermissionError(&err)
// 	return
// }

// // TODO: Thread staff crud

// // CanPerform returns, if the account can perform an action of ModerationLevel
// // 'action' on the target board
// func CanPerform(account, board string, action common.ModerationLevel) (
// 	can bool, err error,
// ) {
// 	switch {
// 	case account == "admin": // admin account can do anything
// 		return true, nil
// 	case action == common.Admin: // Only admin account can perform Admin actions
// 		return false, nil
// 	}

// 	pos, err := FindPosition(board, account)
// 	can = pos >= action
// 	if err == pgx.ErrNoRows {
// 		err = nil
// 	}
// 	return
// }

// // GetSameIPPosts returns posts with the same IP and on the same board as the
// // target post
// func GetSameIPPosts(id uint64, by string) (posts []byte, err error) {
// 	err = db.QueryRow(`get_same_ip_posts`, id, by).Scan(&posts)
// 	castPermissionError(&err)
// 	return
// }

// // Delete posts of the same IP as target post on board and optionally keep
// // deleting posts by this IP
// func DeletePostsByIP(id uint64, account string, keepDeleting time.Duration,
// 	reason string,
// ) (err error) {
// 	seconds := 0
// 	if keepDeleting != 0 {
// 		seconds = int(keepDeleting / time.Second)
// 	}
// 	_, err = db.Exec(
// 		"select delete_posts_by_ip($1::bigint, $2::text, $3::bigint, $4::text)",
// 		id, account, seconds, reason)
// 	castPermissionError(&err)
// 	return
// }

// func castPermissionError(err *error) {
// 	if extractException(*err) == "access denied" {
// 		*err = common.ErrNoPermissions
// 	}
// }

// // DeletePost marks the target post as deleted
// func DeletePosts(ids []uint64, by string) (err error) {
// 	_, err = db.Exec(
// 		"select delete_posts($1::bigint[], $2::text)",
// 		encodeUint64Array(ids),
// 		by,
// 	)
// 	castPermissionError(&err)
// 	return
// }

// // SetThreadLock sets the ability of users to post in a specific thread
// func SetThreadLock(id uint64, locked bool, by string) (err error) {
// 	board, err := GetPostBoard(id)
// 	if err != nil {
// 		return
// 	}
// 	return InTransaction(func(tx *pgx.Tx) (err error) {
// 		_, err = tx.Exec("set_thread_lock", id, locked)
// 		if err != nil {
// 			return
// 		}
// 		return logModeration(tx, auth.ModLogEntry{
// 			ID:    id,
// 			Board: board,
// 			ModerationEntry: common.ModerationEntry{
// 				Type: common.LockThread,
// 				By:   by,
// 				Data: strconv.FormatBool(locked),
// 			},
// 		})
// 	})
// }

// // GetModLog retrieves the moderation log for a specific board
// func GetModLog(board string) (log []byte, err error) {
// 	err = db.QueryRow("get_mod_log", board).Scan(&log)
// 	ensureArray(&log)
// 	return
// }

// // GetModLog retrieves the moderation log entry by ID

// func GetModLogEntry(id uint64) (entry []byte, err error) {
// 	err = db.QueryRow("get_mod_log_entry", id).Scan(&entry)
// 	return
// }
