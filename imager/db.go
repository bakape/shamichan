package imager

import (
	"io"
	"os"

	r "github.com/dancannon/gorethink"

	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/types"
	"github.com/bakape/meguca/util"
)

var (
	// Update associate post count on an image document
	incrementImageRefCount = map[string]r.Term{
		"posts": r.Row.Field("posts").Add(1),
	}
)

// FindImageThumb searches for an existing image with the specified hash and
// returns it, if it exists. Otherwise, returns an empty struct. To ensure the
// image is not deallocated by another theread/process, the refference counter
// of the image will be incremented. If a successfull allocattion is not
// performed, call UnreferenceImage() on this image, to avoid possible dangling
// images.
func FindImageThumb(hash string) (img types.Image, err error) {
	query := db.GetImage(hash).
		Update(incrementImageRefCount, r.UpdateOpts{ReturnChanges: true}).
		Field("changes").
		Field("new_val").
		Default(nil)
	err = db.DB(query).One(&img)
	return
}

type unreferenceResponse struct {
	Posts    int   `gorethink:"posts"`
	FileType uint8 `gorethink:"fileType"`
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

	var res unreferenceResponse
	err := db.DB(query).One(&res)
	if err != nil {
		return err
	}

	if res.Posts == 1 {
		if err := deleteAssets(id, res.FileType); err != nil {
			return err
		}
	}

	return nil
}

// Allocate an image's file resources to their respective served directories and
// write its data to the database
func allocateImage(src, thumb io.Reader, img types.Image) error {
	err := writeAssets(img.SHA1, img.FileType, src, thumb)
	if err != nil {
		return cleanUpFailedAllocation(img, err)
	}

	query := r.
		Table("images").
		Insert(types.ProtoImage{ImageCommon: img.ImageCommon})
	err = db.DB(query).Exec()
	if err != nil {
		return cleanUpFailedAllocation(img, err)
	}
	return nil
}

// Delete any dangling image files in case of a failed image allocattion
func cleanUpFailedAllocation(img types.Image, err error) error {
	delErr := deleteAssets(img.SHA1, img.FileType)
	if err != nil && !os.IsNotExist(delErr) {
		err = util.WrapError(err.Error(), delErr)
	}
	return err
}
