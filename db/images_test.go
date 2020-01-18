package db

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/imager/assets"
	"github.com/bakape/meguca/test"
	"github.com/bakape/meguca/test/test_assets"
	"github.com/jackc/pgx/v4"
)

func prepareSampleImage(t *testing.T) (
	img common.ImageCommon,
	files [2]*os.File,
	close func(),
) {
	t.Helper()

	clearTables(t, "images")
	delDirs := test_assets.SetupImageDirs(t)

	img = common.ImageCommon{
		Width:       300,
		Height:      300,
		ThumbHeight: 150,
		ThumbWidth:  150,
		Size:        1 << 20,
	}
	copy(img.SHA1[:], test.GenBuf(20))
	copy(img.MD5[:], test.GenBuf(16))

	assertNoImage(t, img.SHA1)

	for i, name := range [...]string{"sample", "thumb"} {
		files[i] = test.OpenSample(t, name+".jpg")
	}
	close = func() {
		for _, f := range files {
			f.Close()
		}
		delDirs()
	}
	err := InTransaction(context.Background(), func(tx pgx.Tx) error {
		return AllocateImage(context.Background(), tx, img, files[0], files[1])
	})
	if err != nil {
		t.Fatal(err)
	}

	return
}

func assertNoImage(t *testing.T, id common.SHA1Hash) {
	t.Helper()

	err := InTransaction(context.Background(), func(tx pgx.Tx) (err error) {
		_, err = GetImage(context.Background(), tx, id)
		return
	})
	test.AssertEquals(t, err, pgx.ErrNoRows)
}

func TestAllocateImage(t *testing.T) {
	std, files, close := prepareSampleImage(t)
	defer close()

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

		assertNoImage(t, std.SHA1)
	})
}

func TestInsertImage(t *testing.T) {
	clearTables(t, "threads")
	thread, authKey := insertSampleThread(t)
	img, _, close := prepareSampleImage(t)
	defer close()

	const name = "fuko_da.jpeg"

	err := InTransaction(context.Background(), func(tx pgx.Tx) (err error) {
		return InsertImage(
			context.Background(),
			tx,
			authKey,
			img.SHA1,
			name,
			false,
		)
	})
	if err != nil {
		t.Fatal(err)
	}

	res, err := GetPost(context.Background(), thread)
	if err != nil {
		t.Fatal(err)
	}

	var co struct {
		Created_on int
	}
	err = json.Unmarshal(res, &co)
	if err != nil {
		t.Fatal(err)
	}
	test.AssertJSON(t, bytes.NewReader(res), map[string]interface{}{
		"id":   thread,
		"body": nil,
		"flag": nil,
		"name": nil,
		"open": true,
		"page": 0,
		"sage": false,
		"trip": nil,
		"image": map[string]interface{}{
			"md5":          hex.EncodeToString(img.MD5[:]),
			"name":         "fuko_da.jpeg",
			"sha1":         hex.EncodeToString(img.SHA1[:]),
			"size":         1048576,
			"audio":        false,
			"title":        nil,
			"video":        false,
			"width":        300,
			"artist":       nil,
			"height":       300,
			"duration":     0,
			"file_type":    "JPEG",
			"spoilered":    false,
			"thumb_type":   "JPEG",
			"thumb_width":  150,
			"thumb_height": 150,
		},
		"thread":     thread,
		"created_on": co.Created_on,
	})
}
