package db

import (
	"database/sql"
	"errors"
	"sync"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/Chiiruno/meguca/common"
	"github.com/Chiiruno/meguca/config"
)

// Common errors
var (
	ErrUserNameTaken = errors.New("user name already taken")
	mu               sync.RWMutex
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
	if err == sql.ErrNoRows {
		err = nil
	}
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

// IncrementLoginAttempts increments or creates the login attempt counter for
// user
func IncrementLoginAttempts(ip, account string) (attempts uint8, err error) {
	mu.Lock()
	defer mu.Unlock()
	// Get login attempts counter
	err = sq.Select("attempts").
		From("attempted_logins").
		Where("ip = ? and account = ?", ip, account).
		Scan(&attempts)
	switch err {
	case nil:
		// Don't increment if over 5 attempts
		if attempts > 5 {
			return
		}
		// Increment the counter
		attempts++
		_, err = sq.Update("attempted_logins").
			Set("attempts", attempts).
			Where("ip = ? and account = ?", ip, account).
			Exec()
	case sql.ErrNoRows:
		// If not, create it
		attempts = 1
		_, err = sq.Insert("attempted_logins").
			Columns("ip", "account", "attempts", "expires").
			Values(ip, account, attempts, time.Now().Add(time.Hour*24).UTC()).
			Exec()
	}
	return
}

// ClearLoginAttempts clears login attempts for account for IP
func ClearLoginAttempts(ip, account string) error {
	mu.Lock()
	defer mu.Unlock()
	return clearLoginAttempts(ip, account)
}

func clearLoginAttempts(ip, account string) error {
	_, err := sq.Delete("attempted_logins").
		Where("ip = ? and account = ?", ip, account).
		Exec()
	return err
}

// Decrements all login attempt counts, deletes if under 2
func decrementAllLoginAttempts() (err error) {
	var ip, account string
	var attempts uint8
	mu.Lock()
	defer mu.Unlock()
	r, err := sq.Select("ip", "account", "attempts").
		From("attempted_logins").
		Query()
	if err != nil {
		return
	}
	defer r.Close()
	for r.Next() {
		err = r.Scan(&ip, &account, &attempts)
		if err != nil {
			return
		}
		if attempts < 2 {
			err = clearLoginAttempts(ip, account)
			if err != nil {
				return
			}
			continue
		}
		_, err = sq.Update("attempted_logins").
			Set("attempts", attempts-1).
			Where("ip = ? and account = ?", ip, account).
			Exec()
		if err != nil {
			return
		}
	}
	return r.Err()
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
	returnAll := func() ([]string, error) {
		return append([]string{"all"}, config.GetBoards()...), nil
	}

	// admin account can perform actions on any board
	if account == "admin" {
		return returnAll()
	}

	r, err := sq.Select("board").
		From("staff").
		Where("account = ? and position = ?", account, common.BoardOwner).
		Query()
	if err != nil {
		return
	}
	defer r.Close()

	var board string
	for r.Next() {
		err = r.Scan(&board)
		if err != nil {
			return
		}
		if board == "all" {
			return returnAll()
		}
		boards = append(boards, board)
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
