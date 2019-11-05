// Package server handles client requests for HTML page rendering, JSON and
// websocket connections
package server

import (
	"os"
	"strconv"

	ass "github.com/Chiiruno/meguca/assets"
	"github.com/Chiiruno/meguca/auth"
	"github.com/Chiiruno/meguca/config"
	"github.com/Chiiruno/meguca/db"
	"github.com/Chiiruno/meguca/imager/assets"
	"github.com/Chiiruno/meguca/lang"
	mlog "github.com/Chiiruno/meguca/log"
	"github.com/Chiiruno/meguca/templates"
	"github.com/Chiiruno/meguca/util"
	"github.com/Chiiruno/meguca/websockets/feeds"
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
		f, err = os.OpenFile("errors.log",
			os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
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
	err = lang.Load()
	if err != nil {
		return
	}

	// Depend on configs
	var tasks []func() error
	if config.Server.ImagerMode != config.ImagerOnly {
		tasks = append(tasks, templates.Compile, listenToThreadDeletion)
		go ass.WatchVideoDir()
	}
	if config.Server.ImagerMode != config.NoImager {
		tasks = append(tasks, auth.LoadCaptchaServices)
	}
	tasks = append(tasks, feeds.Init)
	err = util.Parallel(tasks...)
	if err != nil {
		return
	}

	return startWebServer()
}
