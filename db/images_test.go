package db

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/bakape/meguca/imager/assets"
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
		source: filepath.FromSlash("testdata/" + source),
		paths:  assets.GetFilePaths(name, fileType),
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

func (*Tests) TestFindNonexistantImageThumb(c *C) {
	_, err := FindImageThumb("sha")
	c.Assert(err, Equals, r.ErrEmptyResult)
}

func (*Tests) TestFindImageThumb(c *C) {
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
	c.Assert(img, DeepEquals, thumbnailed.ImageCommon)

	assertImageRefCount(id, 2, c)
}

func insertProtoImage(img types.ProtoImage, c *C) {
	c.Assert(Write(r.Table("images").Insert(img)), IsNil)
}

func assertImageRefCount(id string, count int, c *C) {
	var posts int
	c.Assert(One(GetImage(id).Field("posts"), &posts), IsNil)
	c.Assert(posts, Equals, count)
}

func (*Tests) TestDecreaseImageRefCount(c *C) {
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

func (*Tests) TestRemoveUnreffedImage(c *C) {
	const id = "123"
	img := types.ProtoImage{
		ImageCommon: types.ImageCommon{
			FileType: types.JPEG,
			SHA1:     id,
		},
		Posts: 1,
	}
	insertProtoImage(img, c)
	at := newAllocatioTester("sample.jpg", id, types.JPEG, c)
	at.Allocate()

	c.Assert(DeallocateImage(id), IsNil)

	// Assert database document is deleted
	var noImage bool
	c.Assert(One(GetImage(id).Eq(nil), &noImage), IsNil)
	c.Assert(noImage, Equals, true)

	// Assert files are deleted
	at.AssertDeleted()
}

func (*Tests) TestFailedAllocationCleanUp(c *C) {
	const id = "123"
	at := newAllocatioTester("sample.jpg", id, types.JPEG, c)
	at.Allocate()
	c.Assert(os.Remove(filepath.FromSlash("images/thumb/"+id+".jpg")), IsNil)

	err := errors.New("foo")
	img := types.ImageCommon{
		SHA1:     id,
		FileType: types.JPEG,
	}

	c.Assert(cleanUpFailedAllocation(img, err), Equals, err)
	at.AssertDeleted()
}

func (*Tests) TestImageAllocation(c *C) {
	const id = "123"
	var samples [3][]byte
	for i, name := range [...]string{"sample", "thumb"} {
		samples[i] = readSample(name+".jpg", c)
	}
	img := types.ImageCommon{
		SHA1:     id,
		FileType: types.JPEG,
	}

	c.Assert(AllocateImage(samples[0], samples[1], img), IsNil)

	// Assert files and remove them
	for i, path := range assets.GetFilePaths(id, types.JPEG) {
		buf, err := ioutil.ReadFile(path)
		c.Assert(err, IsNil)
		c.Assert(buf, DeepEquals, samples[i])
	}

	// Assert database document
	var imageDoc types.ProtoImage
	c.Assert(One(GetImage(id), &imageDoc), IsNil)
	c.Assert(imageDoc, DeepEquals, types.ProtoImage{
		ImageCommon: img,
		Posts:       1,
	})
}

func readSample(name string, c *C) []byte {
	path := filepath.Join("testdata", name)
	data, err := ioutil.ReadFile(path)
	c.Assert(err, IsNil)
	return data
}

func (*Tests) TestUseImageToken(c *C) {
	const name = "foo.jpeg"
	proto := types.ProtoImage{
		ImageCommon: assets.StdJPEG.ImageCommon,
		Posts:       1,
	}
	c.Assert(Write(r.Table("images").Insert(proto)), IsNil)

	_, id, err := NewImageToken(assets.StdJPEG.SHA1)
	c.Assert(err, IsNil)

	img, err := UseImageToken(id)
	c.Assert(err, IsNil)
	c.Assert(img, DeepEquals, assets.StdJPEG.ImageCommon)
}
