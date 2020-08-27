// Package server handles client requests for HTML page rendering, JSON and
// websocket connections
package server

import (
	"flag"
	"os"
	"strconv"
	"strings"

	"github.com/ErikDubbelboer/gspt"
	ass "github.com/bakape/meguca/assets"
	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/cache"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/imager/assets"
	mlog "github.com/bakape/meguca/log"
	"github.com/bakape/meguca/util"
	"github.com/bakape/meguca/websockets"
)

// Start parses command line arguments and initializes the server.
func Start() (err error) {
	c := &config.Server
	err = c.Load()
	if err != nil {
		return
	}

	flag.StringVar(
		&c.Database,
		"d",
		c.Database,
		"PostgreSQL database URL to connect to",
	)
	flag.Float64Var(
		&c.CacheSize,
		"c",
		c.CacheSize,
		`size limit of internal cache in MB`,
	)
	flag.BoolVar(
		&c.Server.ReverseProxied,
		"r",
		c.Server.ReverseProxied,
		`the server can only be accessed by clients through a reverse proxy and
thus can safely honour "X-Forwarded-For" headers for client IP
resolution`,
	)
	flag.StringVar(
		&c.Server.Address,
		"a",
		c.Server.Address,
		`address to listen on for incoming connections`,
	)
	flag.Parse()

	// Censor DB connection string, if any
	args := make([]string, 0, len(os.Args))
	for i := 0; i < len(os.Args); i++ {
		arg := os.Args[i]
		if strings.HasSuffix(arg, "-d") { // To match both -d and --d
			args = append(args, arg, "****")
			i++ // Jump to args after password
		} else {
			args = append(args, arg)
		}
	}
	gspt.SetProcTitle(strings.Join(args, " "))

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

	err = util.Parallel(db.LoadDB, assets.CreateDirs)
	if err != nil {
		return
	}

	// Depend on configs or DB
	go ass.WatchVideoDir()
	err = util.Parallel(
		cache.Init,
		auth.LoadCaptchaServices,
		websockets.Init,
		func() error {
			mlog.Init(mlog.Console)
			return nil
		},
	)
	if err != nil {
		return
	}

	return startWebServer()
}
