package imager

import (
	"meguca/config"
	"meguca/db"
	"meguca/imager/assets"
	. "meguca/test"
	"os"
	"testing"
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
	t.Helper()
	if err := assets.ResetDirs(); err != nil {
		t.Fatal(err)
	}
}

// How do we assert a thumbnail?
func assertThumbnail(t *testing.T, thumb []byte) {
	t.Helper()
	if thumb != nil && len(thumb) < 100 {
		t.Fatal("thumbnail too small")
	}
}

func assertDims(t *testing.T, res, std [4]uint16) {
	t.Helper()
	if res != std {
		LogUnexpected(t, std, res)
	}
}
