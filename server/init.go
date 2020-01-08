// Package server handles client requests for HTML page rendering, JSON and
// websocket connections
package server

import (
	"os"
	"strconv"

	ass "github.com/bakape/meguca/assets"
	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/cache"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/imager/assets"
	mlog "github.com/bakape/meguca/log"
	"github.com/bakape/meguca/util"
)

// Start parses command line arguments and initializes the server.
func Start() (err error) {
	err = config.Server.Load()
	if err != nil {
		return
	}

	// Write PID file
	f, err := os.Create(".pid")
	if err != nil {
		return
	}
	_, err = f.Write(strconv.AppendInt(nil, int64(os.Getpid()), 10))
	if err != nil {
		return
	}
	err = f.Close()
	if err != nil {
		return
	}

	if !config.Server.Debug {
		var f *os.File
		f, err = os.OpenFile(
			"errors.log",
			os.O_WRONLY|os.O_APPEND|os.O_CREATE,
			0600,
		)
		if err != nil {
			return
		}
		defer f.Close()
		os.Stdout = f
		os.Stderr = f
	}
	mlog.Init(mlog.Console)
	mlog.ConsoleHandler.SetDisplayColor(config.Server.Debug)

	err = util.Parallel(db.LoadDB, assets.CreateDirs)
	if err != nil {
		return
	}

	// Depend on configs or DB
	go ass.WatchVideoDir()
	err = util.Parallel(cache.Init, auth.LoadCaptchaServices, websockets.Init)
	if err != nil {
		return
	}

	return startWebServer()
}
