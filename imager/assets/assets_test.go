package assets

import (
	"bytes"
	"os"
	"testing"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	. "github.com/bakape/meguca/test"
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

func TestGetFilePaths(t *testing.T) {
	t.Parallel()

	webm := GetFilePaths("jingai", common.WEBM, common.PNG)
	jpeg := GetFilePaths("modoki", common.JPEG, common.JPEG)

	cases := [...]struct {
		name, got, expected string
	}{
		{"not JPEG src", webm[0], "images/src/jingai.webm"},
		{"not JPEG thumb", webm[1], "images/thumb/jingai.png"},
		{"JPEG src", jpeg[0], "images/src/modoki.jpg"},
		{"JPEG thumb", jpeg[1], "images/thumb/modoki.jpg"},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			if c.got != c.expected {
				LogUnexpected(t, c.expected, c.got)
			}
		})
	}
}

func TestDeleteAssets(t *testing.T) {
	resetDirs(t)

	cases := [...]struct {
		testName, name      string
		fileType, thumbType uint8
	}{
		{"JPEG", "foo", common.JPEG, common.JPEG},
		{"PNG", "bar", common.PNG, common.PNG},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.testName, func(t *testing.T) {
			t.Parallel()

			// Create files
			for _, path := range GetFilePaths(c.name, c.fileType, c.thumbType) {
				file, err := os.Create(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := file.Close(); err != nil {
					t.Fatal(err)
				}
			}

			// Delete them and check, if deleted
			if err := Delete(c.name, c.fileType, c.thumbType); err != nil {
				t.Fatal(err)
			}
			for _, path := range GetFilePaths(c.name, c.fileType, c.thumbType) {
				_, err := os.Stat(path)
				if !os.IsNotExist(err) {
					UnexpectedError(t, err)
				}
			}
		})
	}
}

func TestDeleteMissingAssets(t *testing.T) {
	resetDirs(t)

	if err := Delete("akarin", common.PNG, common.PNG); err != nil {
		t.Fatal(err)
	}
}

func TestWriteAssets(t *testing.T) {
	resetDirs(t)

	const (
		name      = "foo"
		fileType  = common.JPEG
		thumbType = common.JPEG
	)
	std := [...][]byte{
		{1, 2, 3},
		{4, 5, 6},
	}

	err := Write(name, fileType, thumbType, bytes.NewReader(std[0]),
		bytes.NewReader(std[1]))
	if err != nil {
		t.Fatal(err)
	}

	for i, path := range GetFilePaths(name, fileType, thumbType) {
		AssertFileEquals(t, path, std[i])
	}
}
