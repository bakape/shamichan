package imager

import (
	"os"
	"testing"

	"github.com/bakape/shamichan/imager/assets"
	"github.com/bakape/shamichan/imager/config"
	"github.com/bakape/shamichan/imager/db"
	"github.com/jessevdk/go-flags"
)

func TestMain(m *testing.M) {
	code := 1
	err := func() (err error) {
		_, err = flags.
			NewParser(&config.Server, flags.Default|flags.IgnoreUnknown).
			Parse()
		if err != nil {
			return
		}
		err = db.LoadTestDB()
		if err != nil {
			return
		}

		config.Set(config.Config{
			Public: config.Public{
				Uploads: config.Uploads{
					Max: config.UploadMaximums{
						Height: 2000,
						Width:  2000,
						Size:   20,
					},
				},
			},
		})
		err = assets.CreateDirs()
		if err != nil {
			return
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
