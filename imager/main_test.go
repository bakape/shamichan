package imager

import (
	"os"
	"testing"

	"github.com/bakape/meguca/websockets"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/imager/assets"
)

func TestMain(m *testing.M) {
	code := 1
	err := func() (err error) {
		err = config.Server.Load()
		if err != nil {
			return
		}
		err = db.LoadTestDB()
		if err != nil {
			return
		}

		config.Set(config.Configs{
			MaxHeight: 2000,
			MaxWidth:  2000,
			Public: config.Public{
				MaxSize: 20,
			},
		})
		err = assets.CreateDirs()
		if err != nil {
			return
		}
		err = websockets.Init()
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
