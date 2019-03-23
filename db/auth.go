package db

import (
	"database/sql"
	"errors"
	"time"

	"github.com/Masterminds/squirrel"
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
func RegisterAccount(tx *sql.Tx, id string, hash []byte) error {
	_, err := sq.Insert("accounts").
		Columns("id", "password").
		Values(id, hash).
		RunWith(tx).
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
func FindPosition(board, userID string) (pos common.ModerationLevel, err error) {
	if userID == "admin" {
		return common.Admin, nil
	}

	err = sq.Select("position").
		From("staff").
		Where(squirrel.Eq{
			"account": userID,
			"board":   []string{board, "all"},
		}).
		OrderBy("position desc").
		QueryRow().
		Scan(&pos)
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
		return append([]string{"all"}, config.GetBoards()...), nil
	}

	err = queryAll(
		sq.Select("board").
			From("staff").
			Where("account = ? and position = ?", account, common.BoardOwner),
		func(r *sql.Rows) (err error) {
			var board string
			err = r.Scan(&board)
			if err != nil {
				return
			}
			boards = append(boards, board)
			return
		},
	)
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
