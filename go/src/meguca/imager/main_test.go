package imager

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"meguca/config"
	"meguca/db"
	"meguca/imager/assets"
	. "meguca/test"
)

func TestMain(m *testing.M) {
	db.ConnArgs = db.TestConnArgs
	db.IsTest = true
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
