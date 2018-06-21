package db

import (
	"bytes"
	"database/sql"
	"io/ioutil"
	"meguca/common"
	"meguca/imager/assets"
	. "meguca/test"
	"testing"
	"time"
)

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
	t.Helper()
	if err := WriteImage(assets.StdJPEG.ImageCommon); err != nil {
		t.Fatal(err)
	}
}

func setupImageDirs(t *testing.T) func() {
	t.Helper()
	if err := assets.CreateDirs(); err != nil {
		t.Fatal(err)
	}
	return func() {
		if err := assets.DeleteDirs(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAllocateImage(t *testing.T) {
	assertTableClear(t, "images")
	defer setupImageDirs(t)()

	id := GenString(40)
	var files [2][]byte
	for i, name := range [...]string{"sample", "thumb"} {
		files[i] = ReadSample(t, name+".jpg")
	}
	std := common.ImageCommon{
		SHA1:     id,
		MD5:      GenString(22),
		FileType: common.JPEG,
	}

	if err := AllocateImage(files[0], files[1], std); err != nil {
		t.Fatal(err)
	}

	// Assert files and remove them
	t.Run("files", func(t *testing.T) {
		for i, path := range assets.GetFilePaths(id, common.JPEG, common.JPEG) {
			buf, err := ioutil.ReadFile(path)
			if err != nil {
				t.Error(err)
			}
			if !bytes.Equal(buf, files[i]) {
				t.Error("invalid file")
			}
		}
	})

	// Assert database record
	t.Run("db row", func(t *testing.T) {
		img, err := GetImage(id)
		if err != nil {
			t.Fatal(err)
		}
		if img != std {
			LogUnexpected(t, std, img)
		}
	})
}

func TestImageTokens(t *testing.T) {
	assertTableClear(t, "images")
	writeSampleImage(t)

	token, err := NewImageToken(assets.StdJPEG.SHA1)
	if err != nil {
		t.Fatal(err)
	}

	var img common.ImageCommon
	err = InTransaction(func(tx *sql.Tx) (err error) {
		img, err = UseImageToken(tx, token)
		return
	})
	if err != nil {
		t.Fatal(err)
	}

	std := assets.StdJPEG.ImageCommon
	if img != std {
		LogUnexpected(t, img, std)
	}
}

func TestInsertImage(t *testing.T) {
	assertTableClear(t, "images", "boards")
	writeSampleImage(t)
	writeSampleBoard(t)
	writeSampleThread(t)

	checkHas := func(std bool) {
		has, err := HasImage(1)
		if err != nil {
			t.Fatal(err)
		}
		AssertDeepEquals(t, has, std)
	}

	checkHas(false)

	insertSampleImage(t)
	checkHas(true)

	post, err := GetPost(1)
	if err != nil {
		t.Fatal(err)
	}
	AssertDeepEquals(t, post.Image, &assets.StdJPEG)
}

func insertSampleImage(t *testing.T) {
	err := InTransaction(func(tx *sql.Tx) (err error) {
		return InsertImage(tx, 1, 1, assets.StdJPEG)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSpoilerImage(t *testing.T) {
	assertTableClear(t, "images", "boards")
	writeSampleImage(t)
	writeSampleBoard(t)
	writeSampleThread(t)
	insertSampleImage(t)

	err := SpoilerImage(1, 1)
	if err != nil {
		t.Fatal(err)
	}

	post, err := GetPost(1)
	if err != nil {
		t.Fatal(err)
	}
	AssertDeepEquals(t, post.Image.Spoiler, true)
}

func TestDeleteOwnedImage(t *testing.T) {
	assertTableClear(t, "images", "boards")
	writeSampleImage(t)
	writeSampleBoard(t)
	writeSampleThread(t)
	insertSampleImage(t)

	err := DeleteOwnedImage(1)
	if err != nil {
		t.Fatal(err)
	}

	has, err := HasImage(1)
	if err != nil {
		t.Fatal(err)
	}
	AssertDeepEquals(t, has, false)
}

func TestVideoPlaylist(t *testing.T) {
	std := assets.StdJPEG
	std.FileType = common.WEBM
	std.Audio = true
	std.Video = true
	std.Length = 60

	assertTableClear(t, "images", "boards")
	err := WriteImage(std.ImageCommon)
	if err != nil {
		t.Fatal(err)
	}
	writeSampleBoard(t)
	writeSampleThread(t)
	err = InTransaction(func(tx *sql.Tx) (err error) {
		return InsertImage(tx, 1, 1, std)
	})
	if err != nil {
		t.Fatal(err)
	}

	videos, err := VideoPlaylist("a")
	if err != nil {
		t.Fatal(err)
	}
	AssertDeepEquals(t, videos, []Video{
		{
			FileType: common.WEBM,
			Duration: time.Minute,
			SHA1:     std.SHA1,
		},
	})
}

func TestImageExists(t *testing.T) {
	assertTableClear(t, "images")

	exists, err := ImageExists(assets.StdJPEG.SHA1)
	if err != nil {
		t.Fatal(err)
	}
	AssertDeepEquals(t, exists, false)

	writeSampleImage(t)

	exists, err = ImageExists(assets.StdJPEG.SHA1)
	if err != nil {
		t.Fatal(err)
	}
	AssertDeepEquals(t, exists, true)
}
