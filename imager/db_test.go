package imager

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

type allocationTester struct {
	c            *C
	name, source string
	paths        [2]string
}

func newAllocatioTester(
	source,
	name string,
	fileType uint8,
	c *C,
) *allocationTester {
	return &allocationTester{
		source: filepath.FromSlash("./test/" + source),
		paths:  getFilePaths(name, fileType),
		c:      c,
	}
}

func (a *allocationTester) Allocate() {
	for _, dest := range a.paths {
		a.c.Assert(os.Link(a.source, dest), IsNil)
	}
}

func (a *allocationTester) AssertDeleted() {
	for _, path := range a.paths {
		_, err := os.Stat(path)
		a.c.Assert(err, NotNil)
		a.c.Assert(os.IsNotExist(err), Equals, true)
	}
}

func (*Imager) TestFindNonexistantImageThumb(c *C) {
	_, err := FindImageThumb("sha")
	c.Assert(err, Equals, r.ErrEmptyResult)
}

func (*Imager) TestFindImageThumb(c *C) {
	const id = "foo"
	thumbnailed := types.ProtoImage{
		ImageCommon: types.ImageCommon{
			SHA1: id,
		},
		Posts: 1,
	}
	insertProtoImage(thumbnailed, c)

	img, err := FindImageThumb(id)
	c.Assert(err, IsNil)
	c.Assert(img, DeepEquals, types.Image{
		ImageCommon: thumbnailed.ImageCommon,
	})

	assertImageRefCount(id, 2, c)
}

func insertProtoImage(img types.ProtoImage, c *C) {
	c.Assert(db.Write(r.Table("images").Insert(img)), IsNil)
}

func assertImageRefCount(id string, count int, c *C) {
	var posts int
	c.Assert(db.One(db.GetImage(id).Field("posts"), &posts), IsNil)
	c.Assert(posts, Equals, count)
}

func (*Imager) TestDecreaseImageRefCount(c *C) {
	const id = "123"
	img := types.ProtoImage{
		ImageCommon: types.ImageCommon{
			SHA1: id,
		},
		Posts: 2,
	}
	insertProtoImage(img, c)

	c.Assert(DeallocateImage(id), IsNil)
	assertImageRefCount(id, 1, c)
}

func (*Imager) TestRemoveUnreffedImage(c *C) {
	const id = "123"
	img := types.ProtoImage{
		ImageCommon: types.ImageCommon{
			FileType: jpeg,
			SHA1:     id,
		},
		Posts: 1,
	}
	insertProtoImage(img, c)
	at := newAllocatioTester("sample.jpg", id, jpeg, c)
	at.Allocate()

	c.Assert(DeallocateImage(id), IsNil)

	// Assert database document is deleted
	var noImage bool
	c.Assert(db.One(db.GetImage(id).Eq(nil), &noImage), IsNil)
	c.Assert(noImage, Equals, true)

	// Assert files are deleted
	at.AssertDeleted()
}

func (*Imager) TestFailedAllocationCleanUp(c *C) {
	const id = "123"
	at := newAllocatioTester("sample.jpg", id, jpeg, c)
	at.Allocate()
	c.Assert(os.Remove(filepath.FromSlash("images/thumb/"+id+".jpg")), IsNil)

	err := errors.New("foo")
	img := types.Image{
		ImageCommon: types.ImageCommon{
			SHA1:     id,
			FileType: jpeg,
		},
	}

	c.Assert(cleanUpFailedAllocation(img, err), Equals, err)
	at.AssertDeleted()
}

func (*Imager) TestImageAllocation(c *C) {
	const id = "123"
	var samples [3][]byte
	for i, name := range [...]string{"sample", "thumb"} {
		samples[i] = readSample(name+".jpg", c)
	}
	img := types.Image{
		ImageCommon: types.ImageCommon{
			SHA1:     id,
			FileType: jpeg,
		},
	}

	c.Assert(allocateImage(samples[0], samples[1], img), IsNil)

	// Assert files and remove them
	for i, path := range getFilePaths(id, jpeg) {
		buf, err := ioutil.ReadFile(path)
		c.Assert(err, IsNil)
		c.Assert(buf, DeepEquals, samples[i])
	}

	// Assert database document
	var imageDoc types.ProtoImage
	c.Assert(db.One(db.GetImage(id), &imageDoc), IsNil)
	c.Assert(imageDoc, DeepEquals, types.ProtoImage{
		ImageCommon: img.ImageCommon,
	})
}

func readSample(name string, c *C) []byte {
	path := filepath.FromSlash("test/" + name)
	data, err := ioutil.ReadFile(path)
	c.Assert(err, IsNil)
	return data
}
