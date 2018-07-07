// Package server handles client requests for HTML page rendering, JSON and
// websocket connections
package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"meguca/auth"
	"meguca/cache"
	"meguca/config"
	"meguca/db"
	"meguca/geoip"
	"meguca/imager/assets"
	"meguca/lang"
	"meguca/templates"
	"meguca/util"
	"os"
	"runtime"
	"strings"

	"github.com/ErikDubbelboer/gspt"
	"github.com/go-playground/log"
)

var (
	// debugMode denotes the server has been started with the `debug` parameter.
	// This will cause not to spawn a daemon and stay attached to the launching
	// shell.
	daemonized bool
	isWindows  = runtime.GOOS == "windows"

	// Is assigned in ./daemon.go to control/spawn a daemon process. That file
	// is never compiled on Windows and this function is never called.
	handleDaemon func(string)

	// CLI mode arguments and descriptions
	arguments = map[string]string{
		"start":   "start the meguca server",
		"stop":    "stop a running daemonized meguca server",
		"restart": "combination of stop + start",
		"debug":   "start server in debug mode without daemonizing (default)",
		"help":    "print this help text",
	}
)

// Configs, that can be optionally passed through a JSON configuration file.
// Flags override this. All fields are optional.
type serverConfigs struct {
	SSL, ReverseProxied, Gzip                   *bool
	ImagerMode                                  *uint
	CacheSize                                   *float64
	Address, Database, CertPath, ReverseProxyIP *string
}

func validateImagerMode(m *uint) {
	if *m > 2 {
		panic(fmt.Errorf("invalid imager mode: %d", *m))
	}
}

// Iterate struct fields and assign defaults to missing fields
func setConfigDefaults(c *serverConfigs) {
	if c.SSL == nil {
		c.SSL = new(bool)
	}
	if c.ReverseProxied == nil {
		c.ReverseProxied = new(bool)
	}
	if c.Gzip == nil {
		c.Gzip = new(bool)
	}
	if c.ImagerMode == nil {
		c.ImagerMode = new(uint)
	} else {
		validateImagerMode(c.ImagerMode)
	}
	if c.CacheSize == nil {
		c.CacheSize = new(float64)
		*c.CacheSize = 128
	}
	if c.Address == nil {
		c.Address = new(string)
		*c.Address = "127.0.0.1:8000"
	}
	if c.Database == nil {
		c.Database = new(string)
		*c.Database = db.DefaultConnArgs
	}
	if c.CertPath == nil {
		c.CertPath = new(string)
	}
	if c.ReverseProxyIP == nil {
		c.ReverseProxyIP = new(string)
	}
}

// Start parses command line arguments and initializes the server.
func Start() error {
	// Read config file, if any
	var conf serverConfigs
	buf, err := ioutil.ReadFile("config.json")
	switch {
	case os.IsNotExist(err):
		err = nil
	case err == nil:
		err = json.Unmarshal(buf, &conf)
		if err != nil {
			return err
		}
	default:
		return err
	}
	setConfigDefaults(&conf)

	// Define flags
	flag.StringVar(
		&address,
		"a",
		"127.0.0.1:8000", // Specifying host restricts incoming IP range
		*conf.Address,
	)
	flag.Float64Var(&cache.Size, "c", *conf.CacheSize, "cache size in MB")
	flag.StringVar(
		&db.ConnArgs,
		"d",
		*conf.Database,
		"PostgreSQL connection arguments",
	)
	flag.BoolVar(
		&ssl,
		"s",
		*conf.SSL,
		"serve and listen only through HTTPS. Requires --ssl-cert and "+
			"--ssl-key to be set",
	)
	flag.StringVar(&sslCert, "S", *conf.CertPath, "path to SSL certificate")
	flag.BoolVar(
		&auth.IsReverseProxied,
		"r",
		*conf.ReverseProxied,
		"assume server is behind reverse proxy, when resolving client IPs",
	)
	flag.StringVar(
		&auth.ReverseProxyIP,
		"R",
		*conf.ReverseProxyIP,
		"IP of the reverse proxy. Only needed, when reverse proxy is not on localhost.",
	)
	flag.BoolVar(&enableGzip, "g", *conf.Gzip, "compress all traffic with gzip")
	flag.UintVar(conf.ImagerMode, "i", *conf.ImagerMode,
		`image processing and serving mode for this instance
0	handle image processing and serving and all other functionality (default)
1	handle all functionality except for image processing and serving
2	only handle image processing and serving`)
	flag.Usage = printUsage

	// Parse command line arguments
	flag.Parse()
	if cache.Size < 0 {
		return errors.New("cache size must be a positive number")
	}
	validateImagerMode(conf.ImagerMode)
	config.ImagerMode = config.ImagerModeType(*conf.ImagerMode)
	arg := flag.Arg(0)
	if arg == "" {
		arg = "debug"
	}

	// Can't daemonize in windows, so only args they have is "start" and "help"
	if isWindows {
		switch arg {
		case "debug", "start":
			startServer()
		case "init": // For internal use only
			os.Exit(0)
		default:
			printUsage()
		}
	} else {
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

		handleDaemon(arg)
	}

	return nil
}

// Constructs and prints the CLI help text
func printUsage() {
	os.Stderr.WriteString("Usage: meguca [OPTIONS]... [MODE]\n\nMODES:\n")

	toPrint := []string{"start"}
	if !isWindows {
		toPrint = append(toPrint, []string{"stop", "restart"}...)
	} else {
		arguments["debug"] = `alias of "start"`
	}
	toPrint = append(toPrint, []string{"debug", "help"}...)

	help := new(bytes.Buffer)
	for _, arg := range toPrint {
		fmt.Fprintf(help, "  %s\n    \t%s\n", arg, arguments[arg])
	}

	help.WriteString("\nOPTIONS:\n")
	os.Stderr.Write(help.Bytes())
	flag.PrintDefaults()
	os.Stderr.WriteString(
		"\nConsult the bundled README.md for more information\n",
	)

	os.Exit(1)
}

func startServer() {
	load := func(fns ...func() error) {
		if err := util.Parallel(fns...); err != nil {
			log.Fatal(err)
		}
	}
	load(db.LoadDB, assets.CreateDirs, geoip.Load)
	load(lang.Load, listenToThreadDeletion)
	load(templates.Compile)

	if err := startWebServer(); err != nil {
		log.Fatal(err)
	}
}
