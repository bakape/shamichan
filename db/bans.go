package db

// TODO: Thread-level bans

// var (
// 	// board: IP: IsBanned
// 	banCache = map[string]map[string]bool{}
// 	bansMu   sync.RWMutex
// )

// func writeBan(tx *pgx.Tx, ip net.IP, e auth.ModLogEntry) (err error) {
// 	_, err = tx.Exec("insert_ban", ip, e.Board, e.ID, e.Data, e.By, e.Length)
// 	return
// }

// // Automatically bans an IP
// func SystemBan(ip net.IP, reason string, length time.Duration) (err error) {
// 	return InTransaction(func(tx *pgx.Tx) error {
// 		return systemBanTx(tx, ip, reason, length)
// 	})
// }

// func systemBanTx(
// 	tx *pgx.Tx,
// 	ip net.IP,
// 	reason string,
// 	length time.Duration,
// ) error {
// 	return writeBan(tx, ip, auth.ModLogEntry{
// 		ModerationEntry: common.ModerationEntry{
// 			Type:   common.BanPost,
// 			Data:   reason,
// 			By:     "system",
// 			Length: uint64(length / time.Second),
// 		},
// 		Board: "all",
// 	})
// }

// // Ban IPs from accessing a specific board. Need to target posts. Returns all
// // banned IPs.
// func Ban(id, thread uint64, reason, by string, length time.Duration,) (err error) {
// 	ip, err := GetPostIP(id)
// 	switch err {
// 	case nil:
// 		if ip == nil { // Post already cleared of IP
// 			return
// 		}
// 	case pgx.ErrNoRows:
// 		return nil
// 	default:
// 		return
// 	}

// 	// Write ban messages to posts and ban table
// 	return InTransaction(func(tx *pgx.Tx) (err error) {
// 		return writeBan(tx, ip, auth.ModLogEntry{
// 			ModerationEntry: common.ModerationEntry{
// 				Type:   common.BanPost,
// 				Length: uint64(length / time.Second),
// 				By:     by,
// 				Data:   reason,
// 			},
// 			Board: board,
// 			ID:    id,
// 		})
// 	})
// }

// // Unban lifts a ban from a specific post on a specific board
// func Unban(board string, id uint64, by string) error {
// 	return InTransaction(func(tx *pgx.Tx) (err error) {
// 		_, err = db.Exec("unban", board, id)
// 		if err != nil {
// 			return
// 		}
// 		return logModeration(tx, auth.ModLogEntry{
// 			ModerationEntry: common.ModerationEntry{
// 				Type: common.UnbanPost,
// 				By:   by,
// 			},
// 			Board: board,
// 			ID:    id,
// 		})
// 	})
// }

// func loadBans() (err error) {
// 	err = RefreshBanCache()
// 	if err != nil {
// 		return
// 	}
// 	return Listen(pg_util.ListenOpts{
// 		DebounceInterval: time.Second,
// 		Channel:          "bans.updated",
// 		OnMsg: func(_ string) error {
// 			return RefreshBanCache()
// 		},
// 	})
// }

// // RefreshBanCache loads up to date bans from the database and caches them in
// // memory
// func RefreshBanCache() (err error) {
// 	bans := make([]auth.Ban, 0, 64)

// 	r, err := db.Query("get_bans")
// 	if err != nil {
// 		return
// 	}
// 	defer r.Close()

// 	var b auth.Ban
// 	for r.Next() {
// 		err := r.Scan(&b.IP, &b.Board)
// 		if err != nil {
// 			return err
// 		}
// 		bans = append(bans, b)
// 	}
// 	err = r.Err()
// 	if err != nil {
// 		return
// 	}

// 	new := make(map[string]map[string]bool)
// 	for _, b := range bans {
// 		board, ok := new[b.Board]
// 		if !ok {
// 			board = make(map[string]bool)
// 			new[b.Board] = board
// 		}
// 		board[b.IP.String()] = true
// 	}

// 	bansMu.Lock()
// 	banCache = new
// 	bansMu.Unlock()

// 	return
// }

// // IsBanned checks,  if the IP is banned on the target board or globally
// func IsBanned(board string, ip net.IP) (err error) {
// 	bansMu.RLock()
// 	defer bansMu.RUnlock()

// 	global := banCache["all"]
// 	ips := banCache[board]

// 	ipStr := ip.String()
// 	if (global != nil && global[ipStr]) || (ips != nil && ips[ipStr]) {
// 		// Need to assert ban has not expired and cache is invalid

// 		var banned bool
// 		err = db.QueryRow("is_banned", ip, board).Scan(&banned)
// 		if err != nil {
// 			return
// 		}

// 		if banned {
// 			// Also refresh the cache to keep stale positives from
// 			// triggering a check again
// 			if !common.IsTest {
// 				go func() {
// 					err := RefreshBanCache()
// 					if err != nil {
// 						log.Error(err)
// 					}
// 				}()
// 			}

// 			return common.ErrBanned
// 		}
// 		return
// 	}

// 	return
// }

// // GetBanInfo retrieves information about a specific ban
// func GetBanInfo(ip net.IP, board string) (b auth.BanRecord, err error) {
// 	err = db.
// 		QueryRow("get_ban", ip, board).
// 		Scan(&b.IP, &b.Board, &b.ForPost, &b.Reason, &b.By, &b.Expires)
// 	b.Type = "classic"
// 	return
// }

// // GetBoardBans gets all bans on a specific board. "all" counts as a valid board value.
// func GetBoardBans(board string) (b []auth.BanRecord, err error) {
// 	b = make([]auth.BanRecord, 0, 16)

// 	r, err := db.Query("get_board_bans", board)
// 	if err != nil {
// 		return
// 	}
// 	defer r.Close()

// 	rec := auth.BanRecord{
// 		Ban: auth.Ban{
// 			Board: board,
// 		},
// 	}
// 	for r.Next() {
// 		err = r.Scan(
// 			&rec.IP,
// 			&rec.ForPost,
// 			&rec.Reason,
// 			&rec.By,
// 			&rec.Expires,
// 			&rec.Type,
// 		)
// 		if err != nil {
// 			return
// 		}
// 		b = append(b, rec)
// 	}
// 	err = r.Err()
// 	return
// }
