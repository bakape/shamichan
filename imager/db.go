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

	// Return changes after an update
	returnChanges = r.UpdateOpts{ReturnChanges: true}
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
		Update(incrementImageRefCount, returnChanges).
		Field("changes").
		Field("new_val").
		Default(nil)
	err = db.DB(query).One(&img)
	return
}

// UnreferenceImage decrements the image's refference counter. If the counter
// would become zero, the image entry is immediately deleted. If so, returns
// true.
func UnreferenceImage(id string) (deleted bool, err error) {
	query := db.GetImage(id).
		Replace(func(doc r.Term) r.Term {
			return r.Branch(
				doc.Field("posts").Eq(1),
				nil,
				doc.Merge(map[string]r.Term{
					"posts": doc.Field("posts").Sub(1),
				}),
			)
		}).
		Field("deleted").
		Eq(1).
		Default(false)

	err = db.DB(query).One(&deleted)
	return
}
