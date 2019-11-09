package db

import (
	"errors"
	"net"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/jackc/pgx"
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

	err = db.QueryRow("is_logged_in", user, session).Scan(&loggedIn)
	return
}

// RegisterAccount writes the ID and password hash of a new user account to the
// database
func RegisterAccount(id string, hash []byte) error {
	_, err := db.Exec("register_account", id, hash)
	if IsConflictError(err) {
		return ErrUserNameTaken
	}
	return err
}

// GetPassword retrieves the login password hash of the registered user account
func GetPassword(id string) (hash []byte, err error) {
	err = db.QueryRow("get_password", id).Scan(&hash)
	return
}

// FindPosition returns the highest matching position of a user on a certain
// board. As a special case the admin user will always return "admin".
func FindPosition(
	board, userID string,
) (
	pos common.ModerationLevel,
	err error,
) {
	if userID == "admin" {
		return common.Admin, nil
	}

	err = db.QueryRow("find_position", userID, board).Scan(&pos)
	if err == pgx.ErrNoRows {
		err = nil
	}
	return
}

// WriteLoginSession writes a new user login session to the DB
func WriteLoginSession(account, token string) error {
	_, err := db.Exec("insert_session", account, token)
	return err
}

// LogOut logs the account out of one specific session
func LogOut(account, token string) error {
	_, err := db.Exec("log_out", account, token)
	return err
}

// LogOutAll logs an account out of all user sessions
func LogOutAll(account string) error {
	_, err := db.Exec("log_out_all", account)
	return err
}

// ChangePassword changes an existing user's login password
func ChangePassword(account string, hash []byte) error {
	_, err := db.Exec("change_password", account, hash)
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

	r, err := db.Query("get_owned_boards", account)
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
func GetPostIP(id uint64) (net.IP, error) {
	var s *string
	err := db.QueryRow("get_post_ip", id).Scan(&s)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, nil
	}
	return net.ParseIP(*s), nil
}
