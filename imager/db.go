package imager

import (
	r "github.com/dancannon/gorethink"

	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/types"
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
	query := r.
		Table("images").
		GetAllByIndex("SHA1", hash).
		AtIndex(0).
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

// UnreferenceImage decrements the image's refference counter. If the counter
// would become zero, the image entry is immediately deleted allong with its
// file assets.
func UnreferenceImage(id string) error {
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
