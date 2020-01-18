package db

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/bakape/meguca/test/test_assets"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/imager/assets"
	"github.com/bakape/meguca/test"
	"github.com/jackc/pgx/v4"
)

func TestAllocateImage(t *testing.T) {
	clearTables(t, "images")
	defer test_assets.SetupImageDirs(t)()

	std := common.ImageCommon{
		Width:       300,
		Height:      300,
		ThumbHeight: 150,
		ThumbWidth:  150,
		Size:        1 << 20,
	}
	copy(std.SHA1[:], test.GenBuf(20))
	copy(std.MD5[:], test.GenBuf(16))

	assertNoImage := func(t *testing.T) {
		t.Helper()

		err := InTransaction(context.Background(), func(tx pgx.Tx) (err error) {
			_, err = GetImage(context.Background(), tx, std.SHA1)
			return
		})
		test.AssertEquals(t, err, pgx.ErrNoRows)
	}

	t.Run("no image", assertNoImage)

	var files [2]*os.File
	for i, name := range [...]string{"sample", "thumb"} {
		files[i] = test.OpenSample(t, name+".jpg")
	}
	defer func() {
		for _, f := range files {
			f.Close()
		}
	}()

	err := InTransaction(context.Background(), func(tx pgx.Tx) error {
		return AllocateImage(context.Background(), tx, std, files[0], files[1])
	})
	if err != nil {
		t.Fatal(err)
	}

	// Assert files
	t.Run("files", func(t *testing.T) {
		for i, path := range assets.GetFilePaths(
			std.SHA1,
			common.JPEG,
			common.JPEG,
		) {
			buf, err := ioutil.ReadFile(path)
			if err != nil {
				t.Error(err)
			}

			_, err = files[i].Seek(0, 0)
			if err != nil {
				t.Fatal(err)
			}
			res, err := ioutil.ReadAll(files[i])
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(buf, res) {
				t.Error("invalid file")
			}
		}
	})

	// Assert database record
	t.Run("db row", func(t *testing.T) {
		var img common.ImageCommon
		err := InTransaction(context.Background(), func(tx pgx.Tx) (err error) {
			img, err = GetImage(context.Background(), tx, std.SHA1)
			return
		})
		if err != nil {
			t.Fatal(err)
		}
		test.AssertEquals(t, img, std)
	})

	// Minor cleanup test
	t.Run("delete unused", func(t *testing.T) {
		err := deleteUnusedImages()
		if err != nil {
			t.Fatal(err)
		}

		assertNoImage(t)
	})
}
