package db

import (
	"errors"
	"time"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/dancannon/gorethink"
)

var (
	// ErrUserNameTaken denotes a user name the client is trying  to register
	// with is already taken
	ErrUserNameTaken = errors.New("user name already taken")
)

type banUpdate struct {
	Type             string
	New_val, Old_val struct {
		ID [2]string
	}
}

// IsLoggedIn check if the user is logged in with the specified session
func IsLoggedIn(user, session string) (bool, error) {
	if len(user) > common.MaxLenUserID || len(session) != common.LenSession {
		return false, common.ErrInvalidCreds
	}

	var loggedIn bool
	q := gorethink.
		Table("accounts").
		Get(user).
		Field("sessions").
		Field("token").
		Contains(session).
		Default(false)
	if err := One(q, &loggedIn); err != nil {
		return false, err
	}
	return loggedIn, nil
}

// RegisterAccount writes the ID and password hash of a new user account to the
// database
func RegisterAccount(ID string, hash []byte) error {
	err := Insert("accounts", auth.User{
		ID:       ID,
		Password: hash,
	})
	if gorethink.IsConflictErr(err) {
		return ErrUserNameTaken
	}
	return err
}

// GetLoginHash retrieves the login hash of the registered user account
func GetLoginHash(id string) (hash []byte, err error) {
	query := GetAccount(id).Field("password").Default(nil)
	err = One(query, &hash)
	return
}

// Ban an IP from accessing a specific board
func Ban(rec auth.BanRecord, id uint64) error {
	return WriteAll(
		gorethink.Table("bans").Insert(rec, gorethink.InsertOpts{
			Conflict: "replace",
		}),
		FindPost(id).Update(map[string]interface{}{
			"banned": true,
			"log": gorethink.Row.Field("log").Append(gorethink.
				Expr("13").
				Add(gorethink.Row.Field("id").CoerceTo("string")),
			),
			"lastUpdated": time.Now().Unix(),
		}),
	)
}

// UnBan removes a ban of a specific IP from a specific board
func UnBan(board, ip string) error {
	q := gorethink.
		Table("bans").
		Get([]string{board, ip}).
		Delete().
		Default(nil)
	return Write(q)
}

func loadBans() error {
	cursor, err := gorethink.
		Table("bans").
		Pluck("id").
		Changes(gorethink.ChangesOpts{
			IncludeInitial: true,
			IncludeTypes:   true,
		}).
		Run(RSession)
	if err != nil {
		return err
	}

	ch := make(chan banUpdate)
	cursor.Listen(ch)

	if !IsTest {
		go func() {
			for {
				u := <-ch
				switch u.Type {
				case "initial", "add", "change":
					id := u.New_val.ID
					auth.AddBan(id[0], id[1])
				case "remove":
					id := u.Old_val.ID
					auth.RemoveBan(id[0], id[1])
				}
			}
		}()
	}

	return nil
}
