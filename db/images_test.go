package db

import (
	"bytes"
	"database/sql"
	"io/ioutil"
	"testing"

	"../common"
	"../imager/assets"
	. "../test"
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
	if err := WriteImage(nil, assets.StdJPEG.ImageCommon); err != nil {
		t.Fatal(err)
	}
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

	img, err := UseImageToken(token)
	if err != nil {
		t.Fatal(err)
	}
	std := assets.StdJPEG.ImageCommon
	if img != std {
		LogUnexpected(t, img, std)
	}
}
