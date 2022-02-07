package assets

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	"github.com/bakape/shamichan/imager/common"
	"github.com/bakape/shamichan/imager/config"
	"github.com/bakape/shamichan/imager/test"
)

func init() {
	config.Set(config.Defaults)
}

func TestMain(m *testing.M) {
	if err := CreateDirs(); err != nil {
		panic(err)
	}
	defer DeleteDirs()

	code := m.Run()

	os.Exit(code)
}

func resetDirs(t *testing.T) {
	t.Helper()
	if err := ResetDirs(); err != nil {
		t.Fatal(err)
	}
}

func genID() (id [20]byte, idHex string) {
	copy(id[:], test.GenBuf(20))
	idHex = hex.EncodeToString(id[:])
	return
}

func TestGetFilePaths(t *testing.T) {
	t.Parallel()

	id, idHex := genID()
	webm := GetFilePaths(id, common.WEBM, common.PNG)
	jpeg := GetFilePaths(id, common.JPEG, common.JPEG)

	cases := [...]struct {
		name, got, expected string
	}{
		{
			"not JPEG src",
			webm[0],
			"images/src/%s.webm",
		},
		{
			"not JPEG thumb",
			webm[1],
			"images/thumb/%s.png",
		},
		{
			"JPEG src",
			jpeg[0],
			"images/src/%s.jpg",
		},
		{
			"JPEG thumb",
			jpeg[1],
			"images/thumb/%s.jpg",
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			test.AssertEquals(t, c.got, fmt.Sprintf(c.expected, idHex))
		})
	}
}

func TestDeleteAssets(t *testing.T) {
	resetDirs(t)

	cases := [...]struct {
		testName, name      string
		fileType, thumbType common.FileType
	}{
		{"JPEG", "foo", common.JPEG, common.JPEG},
		{"PNG", "bar", common.PNG, common.PNG},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.testName, func(t *testing.T) {
			t.Parallel()

			id, _ := genID()

			// Create files
			for _, path := range GetFilePaths(id, c.fileType, c.thumbType) {
				file, err := os.Create(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := file.Close(); err != nil {
					t.Fatal(err)
				}
			}

			// Delete them and check, if deleted
			err := Delete(id, c.fileType, c.thumbType)
			if err != nil {
				t.Fatal(err)
			}
			for _, path := range GetFilePaths(id, c.fileType, c.thumbType) {
				_, err := os.Stat(path)
				if !os.IsNotExist(err) {
					test.UnexpectedError(t, err)
				}
			}
		})
	}
}

func TestDeleteMissingAssets(t *testing.T) {
	resetDirs(t)

	id, _ := genID()

	err := Delete(id, common.PNG, common.PNG)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWriteAssets(t *testing.T) {
	resetDirs(t)

	const (
		fileType  = common.JPEG
		thumbType = common.JPEG
	)
	id, _ := genID()
	std := [...][]byte{
		{1, 2, 3},
		{4, 5, 6},
	}

	err := Write(
		id,
		fileType,
		thumbType,
		bytes.NewReader(std[0]),
		bytes.NewReader(std[1]),
	)
	if err != nil {
		t.Fatal(err)
	}

	for i, path := range GetFilePaths(id, fileType, thumbType) {
		test.AssertFileEquals(t, path, std[i])
	}
}

// Archives and such don't generate thumbnails
func TestWriteAssetsNotThumb(t *testing.T) {
	resetDirs(t)

	const (
		fileType  = common.MP3
		thumbType = common.NoFile
	)
	id, _ := genID()
	std := []byte{1, 2, 3}

	err := Write(
		id,
		fileType,
		thumbType,
		bytes.NewReader(std),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	test.AssertFileEquals(t, GetFilePaths(id, fileType, thumbType)[0], std)
}
