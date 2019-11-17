package imager

import (
	"os"
	"testing"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/imager/assets"
	. "github.com/bakape/meguca/test"
)

func TestMain(m *testing.M) {
	code := 1
	err := func() (err error) {
		err = config.Server.Load()
		if err != nil {
			panic(err)
		}
		err = db.LoadTestDB()
		if err != nil {
			panic(err)
		}

		config.Set(config.Configs{})
		err = assets.CreateDirs()
		if err != nil {
			panic(err)
		}

		code = m.Run()
		return assets.DeleteDirs()
	}()
	if err != nil {
		panic(err)
	}
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
