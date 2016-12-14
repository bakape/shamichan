package imager

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/imager/assets"
	. "github.com/bakape/meguca/test"
)

func TestMain(m *testing.M) {
	db.DBName = "meguca_test_imager"
	db.IsTest = true
	assetRoot = filepath.Join("..", "www")
	config.Set(config.Configs{})
	if err := db.LoadDB(); err != nil {
		panic(err)
	}
	if err := assets.CreateDirs(); err != nil {
		panic(err)
	}
	defer assets.DeleteDirs()

	code := m.Run()

	os.Exit(code)
}

func resetDirs(t *testing.T) {
	if err := assets.ResetDirs(); err != nil {
		t.Fatal(err)
	}
}

func readSample(t *testing.T, name string) []byte {
	path := filepath.Join("testdata", name)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func openFile(t *testing.T, name string) *os.File {
	f, err := os.Open(filepath.FromSlash("testdata/" + name))
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// How do we assert a thumbnail?
func assertThumbnail(t *testing.T, thumb []byte) {
	if len(thumb) < 100 {
		t.Fatal("thumbnail too small")
	}
}

func assertDims(t *testing.T, res, std [4]uint16) {
	if res != std {
		LogUnexpected(t, std, res)
	}
}
