package db

import (
	"database/sql"
	"errors"
	"time"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
)

// Common errors
var (
	ErrUserNameTaken = errors.New("user name already taken")
)

// IsLoggedIn check if the user is logged in with the specified session
func IsLoggedIn(user, session string) (loggedIn bool, err error) {
	if len(user) > common.MaxLenUserID || len(session) != common.LenSession {
		err = common.ErrInvalidCreds
		return
	}

	err = prepared["isLoggedIn"].QueryRow(user, session).Scan(&loggedIn)
	if err == sql.ErrNoRows {
		err = nil
	}
	return
}

// RegisterAccount writes the ID and password hash of a new user account to the
// database
func RegisterAccount(ID string, hash []byte) error {
	_, err := db.Exec(
		`INSERT INTO accounts (id, password) VALUES ($1, $2)`,
		ID, hash,
	)
	if IsConflictError(err) {
		return ErrUserNameTaken
	}
	return err
}

// Remove expired login sessions
func expireUserSessions() error {
	_, err := db.Exec(`DELETE FROM sessions WHERE expires < now()`)
	return err
}

// GetPassword retrieves the login password hash of the registered user account
func GetPassword(id string) (hash []byte, err error) {
	err = prepared["getPassword"].QueryRow(id).Scan(&hash)
	return
}

// FindPosition returns the first matching position of a user on a certain
// board. As a special case the admin user will always return "admin". If none
// found, returns empty string
func FindPosition(board, userID string) (pos string, err error) {
	if userID == "admin" {
		return userID, nil
	}
	err = prepared["findPosition"].QueryRow(board, userID).Scan(&pos)
	return
}

// WriteLoginSession writes a new user login session to the DB
func WriteLoginSession(account, token string) error {
	expiryTime := time.Duration(config.Get().SessionExpiry) * time.Hour * 24
	_, err := db.Exec(
		`INSERT INTO sessions (account, token, expires)  VALUES
			($1, $2, $3)`,
		account, token, time.Now().Add(expiryTime),
	)
	return err
}

// LogOut logs the account out of one specific session
func LogOut(account, token string) error {
	_, err := db.Exec(
		`DELETE FROM sessions WHERE account = $1 and token = $2`,
		account, token,
	)
	return err
}

// LogOutAll logs an account out of all user sessions
func LogOutAll(account string) error {
	_, err := db.Exec(`DELETE FROM sessions WHERE account = $1`, account)
	return err
}

// ChangePassword changes an existing user's login password
func ChangePassword(account string, hash []byte) error {
	_, err := db.Exec(
		`UPDATE accounts SET password = $2 WHERE id = $1`,
		account, hash,
	)
	return err
}

// GetPosition returns the staff position a user is holding on a board
func GetPosition(account, board string) (pos string, err error) {
	err = prepared["getPosition"].QueryRow(account, board).Scan(&pos)
	if err == sql.ErrNoRows {
		err = nil
	}
	return
}

// Ban IPs from accessing a specific board. Need to target posts. Returns all
// banned IPs.
func Ban(board, reason, by string, expires time.Time, ids ...uint64) (
	ips map[string]bool, err error,
) {
	tx, err := db.Begin()
	if err != nil {
		return
	}
	defer RollbackOnError(tx, &err)

	// Write ban messages to posts and threads
	q, err := tx.Prepare(
		`UPDATE posts
			SET banned = true
			WHERE id = $1 AND board = $2
			RETURNING ip, op`,
	)
	if err != nil {
		return
	}

	ips = make(map[string]bool, len(ids))
	for _, id := range ids {
		var (
			ip sql.NullString
			op uint64
		)
		err = q.QueryRow(id, board).Scan(&ip, op)
		switch {
		case err != nil:
			return
		case !ip.Valid: // Post older than 7 days
			continue
		}
		ips[ip.String] = true

		var msg []byte
		msg, err = common.EncodeMessage(common.MessageBanned, id)
		if err != nil {
			return
		}
		err = UpdateLog(tx, op, msg)
		if err != nil {
			return
		}
	}

	// Write bans to the ban table
	q, err = tx.Prepare(
		`INSERT INTO bans (ip, board, reason, by, expires) VALUES
			($1, $2, $3, $4, $5)
			ON CONFLICT DO NOTHING`,
	)
	if err != nil {
		return
	}
	for ip := range ips {
		_, err = q.Exec(ip, board, reason, by, expires)
		if err != nil {
			return
		}
	}

	if len(ips) != 0 {
		_, err = tx.Exec(`NOTIFY bans_updated`)
	}
	return
}

func loadBans() error {
	if err := updateBans(); err != nil {
		return err
	}
	return listen("bans_updated", func(_ string) error {
		return updateBans()
	})
}

func updateBans() (err error) {
	r, err := db.Query(`SELECT ip, board FROM bans`)
	if err != nil {
		return
	}
	defer r.Close()

	bans := make([]auth.Ban, 0, 16)
	for r.Next() {
		var b auth.Ban
		err = r.Scan(&b.IP, b.Board)
		if err != nil {
			return
		}
		bans = append(bans, b)
	}
	auth.SetBans(bans...)

	return nil
}

// GetOwnedBoards returns boards the account holder owns
func GetOwnedBoards(account string) (boards []string, err error) {
	r, err := db.Query(
		`SELECT board FROM staff
			WHERE account = $1 AND position = 'owners'`,
		account,
	)
	if err != nil {
		return
	}
	for r.Next() {
		var board string
		err = r.Scan(&board)
		if err != nil {
			return
		}
		boards = append(boards, board)
	}
	return
}

// GetBanInfo retrieves information about a specific ban
func GetBanInfo(ip, board string) (b auth.BanRecord, err error) {
	err = prepared["getBanInfo"].
		QueryRow(ip, board).
		Scan(&b.Board, &b.IP, &b.By, &b.Reason, &b.Expires)
	return
}
