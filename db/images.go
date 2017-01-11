package db

import (
	"database/sql"
	"errors"
	"time"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/lib/pq"
)

const (
	// Time it takes for an image allocation token to expire
	tokenTimeout = time.Minute
)

var (
	// // Update associate post count on an image document
	// incrementImageRefCount = map[string]r.Term{
	// 	"posts": r.Row.Field("posts").Add(1),
	// }

	// ErrInvalidToken occurs, when trying to retrieve an image with an
	// non-existent token. The token might have expired (60 to 119 seconds) or
	// the client could have provided an invalid token to begin with.
	ErrInvalidToken = errors.New("invalid image token")
)

// WriteImage writes a processed image record to the DB
func WriteImage(i common.ImageCommon) error {
	dims := pq.GenericArray{A: i.Dims}
	_, err := prepared["writeImage"].Exec(
		i.APNG, i.Audio, i.Video, i.FileType, i.ThumbType, dims,
		i.Length, i.Size, i.MD5, i.SHA1,
	)
	return err
}

// GetImage retrieves a thumbnailed image record from the DB
func GetImage(SHA1 string) (common.ImageCommon, error) {
	return scanImage(prepared["getImage"].QueryRow(SHA1))
}

func scanImage(rs rowScanner) (img common.ImageCommon, err error) {
	var scanner imageScanner
	err = rs.Scan(scanner.ScanArgs()...)
	if err != nil {
		return
	}
	return scanner.Val().ImageCommon, nil
}

// NewImageToken inserts a new image allocation token into the DB and returns
// it's ID
func NewImageToken(tx *sql.Tx, SHA1 string) (token string, err error) {
	ex := getExecutor(tx, "writeImageToken")

	// Loop in case there is a primary key collision
	for {
		token, err = auth.RandomID(64)
		if err != nil {
			return
		}
		expires := time.Now().Add(tokenTimeout).Unix()

		_, err = ex.Exec(token, SHA1, expires)
		switch {
		case err == nil:
			return
		case isConflictError(err):
			continue
		default:
			return
		}
	}
}

// UseImageToken deletes an image allocation token and returns the matching
// processed image. If no token exists, returns ErrInvalidToken.
func UseImageToken(token string) (img common.ImageCommon, err error) {
	if len(token) != common.LenImageToken {
		err = ErrInvalidToken
		return
	}
	tx, err := db.Begin()
	if err != nil {
		return
	}

	var SHA1 string
	err = tx.Stmt(prepared["useImageToken"]).QueryRow(token).Scan(&SHA1)
	if err != nil {
		tx.Rollback()
		return
	}

	img, err = scanImage(tx.Stmt(prepared["getImage"]).QueryRow(SHA1))
	if err != nil {
		tx.Rollback()
		return
	}
	return img, tx.Commit()
}

// // DeallocateImage decrements the image's reference counter. If the counter
// // would become zero, the image entry is immediately deleted along with its
// // file assets.
// func DeallocateImage(id string) error {
// 	query := GetImage(id).
// 		Replace(
// 			func(doc r.Term) r.Term {
// 				return r.Branch(
// 					doc.Field("posts").Eq(1),
// 					nil,
// 					doc.Merge(map[string]r.Term{
// 						"posts": doc.Field("posts").Sub(1),
// 					}),
// 				)
// 			},
// 			r.ReplaceOpts{ReturnChanges: true},
// 		).
// 		Field("changes").
// 		Field("old_val").
// 		Pluck("posts", "fileType")

// 	var res struct {
// 		Posts               int
// 		FileType, ThumbType uint8
// 	}
// 	if err := One(query, &res); err != nil {
// 		return err
// 	}

// 	if res.Posts == 1 {
// 		if err := assets.Delete(id, res.FileType, res.ThumbType); err != nil {
// 			return err
// 		}
// 	}

// 	return nil
// }

// // AllocateImage allocates an image's file resources to their respective served
// // directories and write its data to the database
// func AllocateImage(src, thumb []byte, img common.ImageCommon) error {
// 	err := assets.Write(img.SHA1, img.FileType, img.ThumbType, src, thumb)
// 	if err != nil {
// 		return cleanUpFailedAllocation(img, err)
// 	}

// 	// TODO: Account for race condition, when the same image is uploaded at the
// 	// same time by multiple clients.
// 	query := r.
// 		Table("images").
// 		Insert(common.ProtoImage{
// 			ImageCommon: img,
// 			Posts:       1,
// 		})
// 	err = Write(query)
// 	if err != nil {
// 		return cleanUpFailedAllocation(img, err)
// 	}
// 	return nil
// }

// // Delete any dangling image files in case of a failed image allocation
// func cleanUpFailedAllocation(img common.ImageCommon, err error) error {
// 	delErr := assets.Delete(img.SHA1, img.FileType, img.ThumbType)
// 	if delErr != nil {
// 		err = util.WrapError(err.Error(), delErr)
// 	}
// 	return err
// }
