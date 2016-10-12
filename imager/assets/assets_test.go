package assets

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/bakape/meguca/test"
	"github.com/bakape/meguca/types"
)

func TestMain(m *testing.M) {
	if err := CreateDirs(); err != nil {
		panic(err)
	}
	defer  DeleteDirs()

	code := m.Run()

	os.Exit(code)
}

func resetDirs(t *testing.T) {
	if err := ResetDirs(); err != nil {
		t.Fatal(err)
	}
}

func TestGetFilePaths(t *testing.T) {
	t.Parallel()

	webm := GetFilePaths("jingai", types.WEBM)
	jpeg := GetFilePaths("modoki", types.JPEG)

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
		testName, name string
		fileType       uint8
	}{
		{"JPEG", "foo", types.JPEG},
		{"PNG", "bar", types.PNG},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.testName, func(t *testing.T) {
			t.Parallel()

			// Create files
			for _, path := range GetFilePaths(c.name, c.fileType) {
				file, err := os.Create(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := file.Close(); err != nil {
					t.Fatal(err)
				}
			}

			// Delete them and check, if deleted
			if err := Delete(c.name, c.fileType); err != nil {
				t.Fatal(err)
			}
			for _, path := range GetFilePaths(c.name, c.fileType) {
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

	if err := Delete("akarin", types.PNG); err != nil {
		t.Fatal(err)
	}
}

func TestWriteFile(t *testing.T) {
	resetDirs(t)

	std := []byte("abc")
	path := filepath.FromSlash("images/src/write_test")
	if err := writeFile(path, std); err != nil {
		t.Fatal(err)
	}

	AssertFileEquals(t, path, std)
}

func TestWriteAssets(t *testing.T) {
	resetDirs(t)

	const (
		name     = "foo"
		fileType = types.JPEG
	)
	std := [...][]byte{
		{1, 2, 3},
		{4, 5, 6},
	}

	if err := Write(name, fileType, std[0], std[1]); err != nil {
		t.Fatal(err)
	}

	for i, path := range GetFilePaths(name, fileType) {
		AssertFileEquals(t, path, std[i])
	}
}
