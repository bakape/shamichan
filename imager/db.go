package imager

import (
	"errors"
	"os"
	"time"

	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/imager/assets"
	"github.com/bakape/meguca/types"
	"github.com/bakape/meguca/util"
	r "github.com/dancannon/gorethink"
)

const (
	// Time it takes for an image allocattion token to expire
	tokenTimeout = time.Minute
)

var (
	// Update associate post count on an image document
	incrementImageRefCount = map[string]r.Term{
		"posts": r.Row.Field("posts").Add(1),
	}

	// ErrInvalidToken occurs, when trying to retrieve an image with an
	// non-existant token. The token might have expired (60 to 119 seconds) or
	// the client could have provided an invalid token to begin with.
	ErrInvalidToken = errors.New("invalid image token")
)

// Cached query
var expireImageTokensQuery = r.
	Table("imageTokens").
	Filter(r.Row.Field("expires").Lt(r.Now())).
	Delete(r.DeleteOpts{ReturnChanges: true}).
	Do(func(d r.Term) r.Term {
		return d.
			Field("deleted").
			Eq(0).
			Branch(
				r.Expr([]string{}),
				d.
					Field("changes").
					Field("old_val").
					Field("SHA1"),
			)
	})

// Document for registering a token coresponding to a client's right to allocate
// an image in its post
type allocationToken struct {
	SHA1    string
	Expires time.Time `gorethink:"expires"`
}

// FindImageThumb searches for an existing image with the specified hash and
// returns it, if it exists. Otherwise, returns an empty struct. To ensure the
// image is not deallocated by another theread/process, the refference counter
// of the image will be incremented.
func FindImageThumb(hash string) (img types.ImageCommon, err error) {
	query := db.GetImage(hash).
		Update(incrementImageRefCount, r.UpdateOpts{ReturnChanges: true}).
		Field("changes").
		Field("new_val").
		Without("posts").
		Default(nil)
	err = db.One(query, &img)
	return
}

// NewImageToken inserts a new expiring image token document into the DB and
// returns it's ID
func NewImageToken(SHA1 string) (code int, token string, err error) {
	q := r.
		Table("imageTokens").
		Insert(allocationToken{
			SHA1:    SHA1,
			Expires: time.Now().Add(tokenTimeout),
		}).
		Field("generated_keys").
		AtIndex(0)
	err = db.One(q, &token)
	if err != nil {
		code = 500
	} else {
		code = 200
	}
	return
}

// UseImageToken deletes a document from the "imageTokens" table and uses and
// returns the Image document from the "images" table, the token was created
// for. If no token exists, returns ErrInvalidToken.
func UseImageToken(id string) (img types.ImageCommon, err error) {
	q := r.
		Table("imageTokens").
		Get(id).
		Delete(r.DeleteOpts{ReturnChanges: true}).
		Field("changes").
		AtIndex(0).
		Field("old_val").
		Pluck("SHA1").
		Merge(r.
			Table("images").
			Get(r.Row.Field("SHA1")).
			Without("posts"),
		).
		Default(nil)
	err = db.One(q, &img)
	if err == r.ErrEmptyResult {
		err = ErrInvalidToken
	}
	return
}

// DeallocateImage decrements the image's refference counter. If the counter
// would become zero, the image entry is immediately deleted allong with its
// file assets.
func DeallocateImage(id string) error {
	query := db.GetImage(id).
		Replace(
			func(doc r.Term) r.Term {
				return r.Branch(
					doc.Field("posts").Eq(1),
					nil,
					doc.Merge(map[string]r.Term{
						"posts": doc.Field("posts").Sub(1),
					}),
				)
			},
			r.ReplaceOpts{ReturnChanges: true},
		).
		Field("changes").
		Field("old_val").
		Pluck("posts", "fileType")

	var res struct {
		Posts    int   `gorethink:"posts"`
		FileType uint8 `gorethink:"fileType"`
	}
	if err := db.One(query, &res); err != nil {
		return err
	}

	if res.Posts == 1 {
		if err := assets.Delete(id, res.FileType); err != nil {
			return err
		}
	}

	return nil
}

// Allocate an image's file resources to their respective served directories and
// write its data to the database
func allocateImage(src, thumb []byte, img types.ImageCommon) error {
	err := assets.Write(img.SHA1, img.FileType, src, thumb)
	if err != nil {
		return cleanUpFailedAllocation(img, err)
	}

	// TODO: Account for race condition, when the same image is uploaded at the
	// same time by multiple clients.
	query := r.
		Table("images").
		Insert(types.ProtoImage{
			ImageCommon: img,
			Posts:       1,
		})
	err = db.Write(query)
	if err != nil {
		return cleanUpFailedAllocation(img, err)
	}
	return nil
}

// Delete any dangling image files in case of a failed image allocattion
func cleanUpFailedAllocation(img types.ImageCommon, err error) error {
	delErr := assets.Delete(img.SHA1, img.FileType)
	if err != nil && !os.IsNotExist(delErr) {
		err = util.WrapError(err.Error(), delErr)
	}
	return err
}

// Remove any expired image tokens and decrement or dealocate their target
// image's assets
func expireImageTokens() error {
	var toDealloc []string
	if err := db.All(expireImageTokensQuery, &toDealloc); err != nil {
		return err
	}

	for _, sha1 := range toDealloc {
		if err := DeallocateImage(sha1); err != nil {
			return err
		}
	}

	return nil
}
