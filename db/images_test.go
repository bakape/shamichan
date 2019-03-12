package db

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/imager/assets"
	"github.com/bakape/meguca/test"
	"os"
	"sync"
	"testing"
	"time"
)

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
	cleanUp := setupImageDirs(t)

	var wg sync.WaitGroup
	wg.Add(3)
	id := test.GenString(40)
	var files [2]*os.File
	for i, name := range [...]string{"sample", "thumb"} {
		files[i] = test.OpenSample(t, name+".jpg")
	}
	std := common.ImageCommon{
		SHA1:     id,
		MD5:      test.GenString(22),
		FileType: common.JPEG,
	}

	err := InTransaction(false, func(tx *sql.Tx) error {
		return AllocateImage(tx, files[0], files[1], std)
	})
	if err != nil {
		t.Fatal(err)
	}

	// Assert files and remove them
	t.Run("files", func(t *testing.T) {
		t.Parallel()
		defer wg.Done()

		defer func() {
			for _, f := range files {
				f.Close()
			}
		}()

		for i, path := range assets.GetFilePaths(id, common.JPEG, common.JPEG) {
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
		t.Parallel()
		defer wg.Done()

		var buf []byte
		err := sq.Select("to_jsonb(i)").
			From("images i").
			Where("sha1 = ?", id).
			QueryRow().
			Scan(&buf)
		if err != nil {
			t.Fatal(err)
		}
		var img common.ImageCommon
		err = json.Unmarshal(buf, &img)
		if err != nil {
			t.Fatal(err)
		}
		if img != std {
			test.LogUnexpected(t, std, img)
		}
	})

	t.Run("get image", func(t *testing.T) {
		t.Parallel()
		defer wg.Done()

		_, err := GetImage(id)
		if err != nil {
			t.Fatal(err)
		}
	})

	// Minor cleanup test
	t.Run("delete unused", func(t *testing.T) {
		t.Parallel()
		wg.Wait()
		defer cleanUp()

		err := deleteUnusedImages()
		if err != nil {
			t.Fatal(err)
		}
		var exists bool
		err = InTransaction(false, func(tx *sql.Tx) (err error) {
			exists, err = ImageExists(tx, id)
			return
		})
		if err != nil {
			t.Fatal(err)
		}
		if exists {
			t.Fatal("image not deleted")
		}
	})
}

func newImageToken(t *testing.T, sha1 string) (token string) {
	t.Helper()

	err := InTransaction(false, func(tx *sql.Tx) (err error) {
		token, err = NewImageToken(tx, sha1)
		return
	})
	if err != nil {
		t.Fatal(err)
	}
	return
}

func TestInsertImage(t *testing.T) {
	assertTableClear(t, "images", "boards")
	prepareThreads(t)
	token := newImageToken(t, assets.StdJPEG.SHA1)
	const postID = 3

	checkHas := func(std bool) {
		has, err := HasImage(postID)
		if err != nil {
			t.Fatal(err)
		}
		test.AssertDeepEquals(t, has, std)
	}

	checkHas(false)

	std := assets.StdJPEG
	var buf []byte

	insert := func() error {
		return InTransaction(false, func(tx *sql.Tx) (err error) {
			buf, err = InsertImage(tx, postID, token, std.Name, std.Spoiler)
			return
		})
	}

	err := insert()
	if err != nil {
		t.Fatal(err)
	}
	checkHas(true)

	type result struct {
		common.Image
		ID uint64
	}

	var img result
	err = json.Unmarshal(buf, &img)
	if err != nil {
		t.Fatal(err)
	}
	test.AssertDeepEquals(t, img, result{
		ID:    postID,
		Image: std,
	})

	// Test token is properly expended
	err = insert()
	if err != ErrInvalidToken {
		t.Fatal(err)
	}
}

func insertSampleImage(t *testing.T) {
	t.Helper()

	token := newImageToken(t, assets.StdJPEG.SHA1)
	err := InTransaction(false, func(tx *sql.Tx) (err error) {
		std := assets.StdJPEG
		_, err = InsertImage(tx, 1, token, std.Name, std.Spoiler)
		return
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
	test.AssertDeepEquals(t, post.Image.Spoiler, true)
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
	token := newImageToken(t, std.SHA1)
	err = InTransaction(false, func(tx *sql.Tx) (err error) {
		_, err = InsertImage(tx, 1, token, std.Name, std.Spoiler)
		return
	})
	if err != nil {
		t.Fatal(err)
	}

	videos, err := VideoPlaylist("a")
	if err != nil {
		t.Fatal(err)
	}
	test.AssertDeepEquals(t, videos, []Video{
		{
			FileType: common.WEBM,
			Duration: time.Minute,
			SHA1:     std.SHA1,
		},
	})
}

func TestImageExists(t *testing.T) {
	assertTableClear(t, "images")

	var exists bool
	err := InTransaction(false, func(tx *sql.Tx) (err error) {
		exists, err = ImageExists(tx, assets.StdJPEG.SHA1)
		return
	})
	if err != nil {
		t.Fatal(err)
	}
	test.AssertDeepEquals(t, exists, false)

	writeSampleImage(t)

	err = InTransaction(false, func(tx *sql.Tx) (err error) {
		exists, err = ImageExists(tx, assets.StdJPEG.SHA1)
		return
	})
	if err != nil {
		t.Fatal(err)
	}
	test.AssertDeepEquals(t, exists, true)
}
