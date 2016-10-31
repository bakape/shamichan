package db

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"bytes"

	"github.com/bakape/meguca/imager/assets"
	. "github.com/bakape/meguca/test"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
)

type allocationTester struct {
	t            *testing.T
	name, source string
	paths        [2]string
}

func newAllocationTester(
	t *testing.T,
	source,
	name string,
	fileType uint8,
) *allocationTester {
	return &allocationTester{
		source: filepath.Join("testdata", source),
		paths:  assets.GetFilePaths(name, fileType),
		t:      t,
	}
}

func (a *allocationTester) Allocate() {
	for _, dest := range a.paths {
		if err := os.Link(a.source, dest); err != nil {
			a.t.Fatal(err)
		}
	}
}

func (a *allocationTester) AssertDeleted() {
	for _, path := range a.paths {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			UnexpectedError(a.t, err)
		}
	}
}

func assertImageRefCount(t *testing.T, id string, count int) {
	var posts int
	if err := One(GetImage(id).Field("posts"), &posts); err != nil {
		t.Fatal(err)
	}
	if posts != count {
		t.Errorf("unexpected reference count: %d : %d", count, posts)
	}
}

func TestFindImageThumb(t *testing.T) {
	assertTableClear(t, "images")

	t.Run("nonexistent image", func(t *testing.T) {
		t.Parallel()
		_, err := FindImageThumb("sha")
		if err != r.ErrEmptyResult {
			UnexpectedError(t, err)
		}
	})
	t.Run("existent image", testFindImageThumb)
}

func testFindImageThumb(t *testing.T) {
	t.Parallel()

	const id = "foo"
	thumbnailed := types.ProtoImage{
		ImageCommon: types.ImageCommon{
			SHA1: id,
		},
		Posts: 1,
	}
	assertInsert(t, "images", thumbnailed)

	img, err := FindImageThumb(id)
	if err != nil {
		t.Fatal(err)
	}
	AssertDeepEquals(t, img, thumbnailed.ImageCommon)
	assertImageRefCount(t, id, 2)
}

func TestDeallocateImage(t *testing.T) {
	assertTableClear(t, "posts", "images")

	t.Run("only decrement ref count", testDecrementRefCount)
	t.Run("remove image", testRemoveImage)
}

func testDecrementRefCount(t *testing.T) {
	t.Parallel()

	const id = "nano desu"
	assertInsert(t, "images", types.ProtoImage{
		ImageCommon: types.ImageCommon{
			SHA1: id,
		},
		Posts: 2,
	})

	if err := DeallocateImage(id); err != nil {
		t.Fatal(err)
	}

	assertImageRefCount(t, id, 1)
}

func testRemoveImage(t *testing.T) {
	t.Parallel()
	defer setupImageDirs(t)()

	const id = "fuwa fuwa fuwari"
	assertInsert(t, "images", types.ProtoImage{
		ImageCommon: types.ImageCommon{
			FileType: types.JPEG,
			SHA1:     id,
		},
		Posts: 1,
	})
	at := newAllocationTester(t, "sample.jpg", id, types.JPEG)
	at.Allocate()

	if err := DeallocateImage(id); err != nil {
		t.Fatal(err)
	}

	assertDeleted(t, GetImage(id), true)
	at.AssertDeleted()
}

func setupImageDirs(t *testing.T) func() {
	if err := assets.CreateDirs(); err != nil {
		t.Fatal(err)
	}
	return func() {
		if err := assets.DeleteDirs(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCleanUpFailedAllocation(t *testing.T) {
	defer setupImageDirs(t)()

	const id = "123"
	at := newAllocationTester(t, "sample.jpg", id, types.JPEG)
	at.Allocate()
	path := filepath.Join("images", "thumb", id+".jpg")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}

	err := errors.New("foo")
	img := types.ImageCommon{
		SHA1:     id,
		FileType: types.JPEG,
	}

	if reErr := cleanUpFailedAllocation(img, err); reErr != err {
		LogUnexpected(t, err, reErr)
	}
	at.AssertDeleted()
}

func TestAllocateImage(t *testing.T) {
	assertTableClear(t, "images")
	defer setupImageDirs(t)()

	const id = "123"
	var files [2][]byte
	for i, name := range [...]string{"sample", "thumb"} {
		files[i] = readSample(t, name+".jpg")
	}
	img := types.ImageCommon{
		SHA1:     id,
		FileType: types.JPEG,
	}

	if err := AllocateImage(files[0], files[1], img); err != nil {
		t.Fatal(err)
	}

	// Assert files and remove them
	t.Run("files", func(t *testing.T) {
		for i, path := range assets.GetFilePaths(id, types.JPEG) {
			buf, err := ioutil.ReadFile(path)
			if err != nil {
				t.Error(err)
			}
			if !bytes.Equal(buf, files[i]) {
				t.Error("invalid file")
			}
		}
	})

	// Assert database document
	t.Run("db document", func(t *testing.T) {
		var doc types.ProtoImage
		if err := One(GetImage(id), &doc); err != nil {
			t.Fatal(err)
		}
		std := types.ProtoImage{
			ImageCommon: img,
			Posts:       1,
		}
		if doc != std {
			LogUnexpected(t, std, doc)
		}
	})
}

func readSample(t *testing.T, name string) []byte {
	path := filepath.Join("testdata", name)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Error(err)
	}
	return data
}

func TestUseImageToken(t *testing.T) {
	assertTableClear(t, "images", "imageTokens")

	const name = "foo.jpeg"
	assertInsert(t, "images", types.ProtoImage{
		ImageCommon: assets.StdJPEG.ImageCommon,
		Posts:       1,
	})

	_, id, err := NewImageToken(assets.StdJPEG.SHA1)
	if err != nil {
		t.Fatal(err)
	}

	img, err := UseImageToken(id)
	if err != nil {
		t.Fatal(err)
	}
	std := assets.StdJPEG.ImageCommon
	if img != std {
		LogUnexpected(t, img, std)
	}
}
