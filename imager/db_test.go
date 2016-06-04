package imager

import (
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

func (*Imager) TestFindNonexistantImageThumb(c *C) {
	img, err := FindImageThumb("sha")
	c.Assert(err, IsNil)
	c.Assert(img, DeepEquals, types.Image{})
}

func (*Imager) TestFindImageThumb(c *C) {
	thumbnailed := types.ProtoImage{
		ImageCommon: types.ImageCommon{
			File: "123",
			SHA1: "foo",
		},
		Posts: 1,
	}
	insertProtoImage(thumbnailed, c)

	img, err := FindImageThumb("foo")
	c.Assert(err, IsNil)
	c.Assert(img, DeepEquals, types.Image{
		ImageCommon: thumbnailed.ImageCommon,
	})

	assertImageRefCount("123", 2, c)
}

func insertProtoImage(img types.ProtoImage, c *C) {
	c.Assert(db.DB(r.Table("images").Insert(img)).Exec(), IsNil)
}

func assertImageRefCount(id string, count int, c *C) {
	var posts int
	c.Assert(db.DB(db.GetImage(id).Field("posts")).One(&posts), IsNil)
	c.Assert(posts, Equals, count)
}

func (*Imager) TestDecreaseImageRefCount(c *C) {
	const id = "123"
	img := types.ProtoImage{
		ImageCommon: types.ImageCommon{
			File: id,
		},
		Posts: 2,
	}
	insertProtoImage(img, c)

	c.Assert(UnreferenceImage(id), IsNil)
	assertImageRefCount(id, 1, c)
}

func (*Imager) TestRemoveUnreffedImage(c *C) {
	const id = "123"
	img := types.ProtoImage{
		ImageCommon: types.ImageCommon{
			File: id,
		},
		Posts: 1,
	}
	insertProtoImage(img, c)

	c.Assert(UnreferenceImage(id), IsNil)

	var noImage bool
	c.Assert(db.DB(db.GetImage(id).Eq(nil)).One(&noImage), IsNil)
	c.Assert(noImage, Equals, true)
}
