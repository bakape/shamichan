package db

import (
	"database/sql"
	"testing"

	"github.com/bakape/meguca/imager/assets"
	. "github.com/bakape/meguca/test"
)

// type allocationTester struct {
// 	t            *testing.T
// 	name, source string
// 	paths        [2]string
// }

// func newAllocationTester(
// 	t *testing.T,
// 	source,
// 	name string,
// 	fileType, thumbType uint8,
// ) *allocationTester {
// 	return &allocationTester{
// 		source: filepath.Join("testdata", source),
// 		paths:  assets.GetFilePaths(name, fileType, thumbType),
// 		t:      t,
// 	}
// }

// func (a *allocationTester) Allocate() {
// 	for _, dest := range a.paths {
// 		if err := os.Link(a.source, dest); err != nil {
// 			a.t.Fatal(err)
// 		}
// 	}
// }

// func (a *allocationTester) AssertDeleted() {
// 	for _, path := range a.paths {
// 		if _, err := os.Stat(path); !os.IsNotExist(err) {
// 			UnexpectedError(a.t, err)
// 		}
// 	}
// }

// func assertImageRefCount(t *testing.T, id string, count int) {
// 	var posts int
// 	if err := One(GetImage(id).Field("posts"), &posts); err != nil {
// 		t.Fatal(err)
// 	}
// 	if posts != count {
// 		t.Errorf("unexpected reference count: %d : %d", count, posts)
// 	}
// }

func TestGetImage(t *testing.T) {
	assertTableClear(t, "images")
	writeSampleImage(t)

	t.Run("nonexistent", func(t *testing.T) {
		t.Parallel()
		_, err := GetImage(GenString(40))
		if err != sql.ErrNoRows {
			UnexpectedError(t, err)
		}
	})
	t.Run("existent", func(t *testing.T) {
		t.Parallel()

		img, err := GetImage(assets.StdJPEG.SHA1)
		if err != nil {
			t.Fatal(err)
		}
		AssertDeepEquals(t, img, assets.StdJPEG.ImageCommon)
	})
}

func writeSampleImage(t *testing.T) {
	if err := WriteImage(assets.StdJPEG.ImageCommon); err != nil {
		t.Fatal(err)
	}
}

// func TestDeallocateImage(t *testing.T) {
// 	assertTableClear(t, "posts", "images")

// 	t.Run("only decrement ref count", testDecrementRefCount)
// 	t.Run("remove image", testRemoveImage)
// }

// func testDecrementRefCount(t *testing.T) {
// 	t.Parallel()

// 	const id = "nano desu"
// 	assertInsert(t, "images", common.ProtoImage{
// 		ImageCommon: common.ImageCommon{
// 			SHA1: id,
// 		},
// 		Posts: 2,
// 	})

// 	if err := DeallocateImage(id); err != nil {
// 		t.Fatal(err)
// 	}

// 	assertImageRefCount(t, id, 1)
// }

// func testRemoveImage(t *testing.T) {
// 	t.Parallel()
// 	defer setupImageDirs(t)()

// 	const id = "fuwa fuwa fuwari"
// 	assertInsert(t, "images", common.ProtoImage{
// 		ImageCommon: common.ImageCommon{
// 			FileType: common.JPEG,
// 			SHA1:     id,
// 		},
// 		Posts: 1,
// 	})
// 	at := newAllocationTester(t, "sample.jpg", id, common.JPEG, common.JPEG)
// 	at.Allocate()

// 	if err := DeallocateImage(id); err != nil {
// 		t.Fatal(err)
// 	}

// 	assertDeleted(t, GetImage(id), true)
// 	at.AssertDeleted()
// }

// func setupImageDirs(t *testing.T) func() {
// 	if err := assets.CreateDirs(); err != nil {
// 		t.Fatal(err)
// 	}
// 	return func() {
// 		if err := assets.DeleteDirs(); err != nil {
// 			t.Fatal(err)
// 		}
// 	}
// }

// func TestCleanUpFailedAllocation(t *testing.T) {
// 	defer setupImageDirs(t)()

// 	const id = "123"
// 	at := newAllocationTester(t, "sample.jpg", id, common.JPEG, common.JPEG)
// 	at.Allocate()
// 	path := filepath.Join("images", "thumb", id+".jpg")
// 	if err := os.Remove(path); err != nil {
// 		t.Fatal(err)
// 	}

// 	err := errors.New("foo")
// 	img := common.ImageCommon{
// 		SHA1:     id,
// 		FileType: common.JPEG,
// 	}

// 	if reErr := cleanUpFailedAllocation(img, err); reErr != err {
// 		LogUnexpected(t, err, reErr)
// 	}
// 	at.AssertDeleted()
// }

// func TestAllocateImage(t *testing.T) {
// 	assertTableClear(t, "images")
// 	defer setupImageDirs(t)()

// 	const id = "123"
// 	var files [2][]byte
// 	for i, name := range [...]string{"sample", "thumb"} {
// 		files[i] = readSample(t, name+".jpg")
// 	}
// 	img := common.ImageCommon{
// 		SHA1:     id,
// 		FileType: common.JPEG,
// 	}

// 	if err := AllocateImage(files[0], files[1], img); err != nil {
// 		t.Fatal(err)
// 	}

// 	// Assert files and remove them
// 	t.Run("files", func(t *testing.T) {
// 		for i, path := range assets.GetFilePaths(id, common.JPEG, common.JPEG) {
// 			buf, err := ioutil.ReadFile(path)
// 			if err != nil {
// 				t.Error(err)
// 			}
// 			if !bytes.Equal(buf, files[i]) {
// 				t.Error("invalid file")
// 			}
// 		}
// 	})

// 	// Assert database document
// 	t.Run("db document", func(t *testing.T) {
// 		var doc common.ProtoImage
// 		if err := One(GetImage(id), &doc); err != nil {
// 			t.Fatal(err)
// 		}
// 		std := common.ProtoImage{
// 			ImageCommon: img,
// 			Posts:       1,
// 		}
// 		if doc != std {
// 			LogUnexpected(t, std, doc)
// 		}
// 	})
// }

// func readSample(t *testing.T, name string) []byte {
// 	path := filepath.Join("testdata", name)
// 	data, err := ioutil.ReadFile(path)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	return data
// }

func TestImageTokens(t *testing.T) {
	assertTableClear(t, "images")
	writeSampleImage(t)

	token, err := NewImageToken(nil, assets.StdJPEG.SHA1)
	if err != nil {
		t.Fatal(err)
	}

	img, err := UseImageToken(token)
	if err != nil {
		t.Fatal(err)
	}
	std := assets.StdJPEG.ImageCommon
	if img != std {
		LogUnexpected(t, img, std)
	}
}
