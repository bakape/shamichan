package db

import (
	"database/sql"
	"errors"
	"meguca/auth"
	"meguca/common"
	"meguca/config"
	"time"

	"github.com/Masterminds/squirrel"
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

	err = sq.Select("true").
		From("sessions").
		Where("account = ? and token = ?", user, session).
		QueryRow().
		Scan(&loggedIn)
	if err == sql.ErrNoRows {
		err = nil
	}
	return
}

// RegisterAccount writes the ID and password hash of a new user account to the
// database
func RegisterAccount(id string, hash []byte) error {
	_, err := sq.Insert("accounts").
		Columns("id", "password").
		Values(id, hash).
		Exec()
	if IsConflictError(err) {
		return ErrUserNameTaken
	}
	return err
}

// GetPassword retrieves the login password hash of the registered user account
func GetPassword(id string) (hash []byte, err error) {
	err = sq.Select("password").
		From("accounts").
		Where("id = ?", id).
		QueryRow().
		Scan(&hash)
	return
}

// FindPosition returns the highest matching position of a user on a certain
// board. As a special case the admin user will always return "admin".
func FindPosition(board, userID string) (pos auth.ModerationLevel, err error) {
	if userID == "admin" {
		return auth.Admin, nil
	}

	r, err := sq.Select("position").
		From("staff").
		Where("account = ? and board = ?", userID, board).
		Query()
	if err != nil {
		return
	}
	defer r.Close()

	// Read the highest position held
	var s string
	for r.Next() {
		err = r.Scan(&s)
		if err != nil {
			return
		}

		level := auth.NotStaff
		switch s {
		case "owners":
			level = auth.BoardOwner
		case "moderators":
			level = auth.Moderator
		case "janitors":
			level = auth.Janitor
		}
		if level > pos {
			pos = level
		}
	}
	err = r.Err()
	return
}

// WriteLoginSession writes a new user login session to the DB
func WriteLoginSession(account, token string) error {
	expiryTime := time.Duration(config.Get().SessionExpiry) * time.Hour * 24
	_, err := sq.Insert("sessions").
		Columns("account", "token", "expires").
		Values(account, token, time.Now().Add(expiryTime)).
		Exec()
	return err
}

// LogOut logs the account out of one specific session
func LogOut(account, token string) error {
	_, err := sq.Delete("sessions").
		Where("account = ? and token = ?", account, token).
		Exec()
	return err
}

// LogOutAll logs an account out of all user sessions
func LogOutAll(account string) error {
	_, err := sq.Delete("sessions").
		Where("account = ?", account).
		Exec()
	return err
}

// ChangePassword changes an existing user's login password
func ChangePassword(account string, hash []byte) error {
	_, err := sq.Update("accounts").
		Set("password", hash).
		Where("id = ?", account).
		Exec()
	return err
}

// GetOwnedBoards returns boards the account holder owns
func GetOwnedBoards(account string) (boards []string, err error) {
	// admin account can perform actions on any board
	if account == "admin" {
		return config.GetBoards(), nil
	}

	r, err := sq.Select("board").
		From("staff").
		Where("account = ? and position = 'owners'", account).
		Query()
	if err != nil {
		return
	}
	defer r.Close()
	for r.Next() {
		var board string
		err = r.Scan(&board)
		if err != nil {
			return
		}
		boards = append(boards, board)
	}
	err = r.Err()
	return
}

func getBans() squirrel.SelectBuilder {
	return sq.Select("ip", "board", "forPost", "reason", "by", "expires").
		From("bans")
}

func scanBanRecord(rs rowScanner) (b auth.BanRecord, err error) {
	err = rs.Scan(&b.IP, &b.Board, &b.ForPost, &b.Reason, &b.By, &b.Expires)
	return
}

// GetBanInfo retrieves information about a specific ban
func GetBanInfo(ip, board string) (auth.BanRecord, error) {
	r := getBans().
		Where("ip = ? and board = ? and expires >= now()", ip, board).
		QueryRow()
	return scanBanRecord(r)
}

// Get all bans on a specific board. "all" counts as a valid board value.
func GetBoardBans(board string) (b []auth.BanRecord, err error) {
	b = make([]auth.BanRecord, 0, 64)

	r, err := getBans().
		Where("board = ? and expires >= now()", board).
		Query()
	if err != nil {
		return
	}
	defer r.Close()

	var rec auth.BanRecord
	for r.Next() {
		rec, err = scanBanRecord(r)
		if err != nil {
			return
		}
		rec.Board = board
		b = append(b, rec)
	}
	err = r.Err()

	return
}

// GetIP returns an IP of the poster that created a post. Posts older than 7
// days will not have this information.
func GetIP(id uint64) (string, error) {
	var ip sql.NullString
	err := sq.Select("ip").
		From("posts").
		Where("id = ?", id).
		QueryRow().
		Scan(&ip)
	return ip.String, err
}
