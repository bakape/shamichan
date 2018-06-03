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
	"meguca/db"
	"meguca/geoip"
	"meguca/imager/assets"
	"meguca/lang"
	"meguca/templates"
	"meguca/util"
	"os"
	"runtime"

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
	CacheSize                                   *float64
	Address, Database, CertPath, ReverseProxyIP *string
}

var defaultServerConfigs = serverConfigs{
	SSL:            newBool(false),
	ReverseProxied: newBool(false),
	Gzip:           newBool(false),
	CacheSize:      newFloat(128),
	Address:        newString("127.0.0.1:8000"),
	Database:       newString(db.DefaultConnArgs),
	CertPath:       newString(""),
	ReverseProxyIP: newString(""),
}

// Creates a heap pointer to a bool
func newBool(b bool) *bool {
	return &b
}

// Creates a heap pointer to a uint
func newFloat(i float64) *float64 {
	return &i
}

// Creates a heap pointer to a uint
func newString(s string) *string {
	return &s
}

// Iterate struct fields and assign defaults to missing fields
func setConfigDefaults(c *serverConfigs) {
	d := defaultServerConfigs
	if c.SSL == nil {
		c.SSL = d.SSL
	}
	if c.ReverseProxied == nil {
		c.ReverseProxied = d.ReverseProxied
	}
	if c.Gzip == nil {
		c.Gzip = d.Gzip
	}
	if c.CacheSize == nil {
		c.CacheSize = d.CacheSize
	}
	if c.Address == nil {
		c.Address = d.Address
	}
	if c.Database == nil {
		c.Database = d.Database
	}
	if c.CertPath == nil {
		c.CertPath = d.CertPath
	}
	if c.ReverseProxyIP == nil {
		c.ReverseProxyIP = d.ReverseProxyIP
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
		setConfigDefaults(&conf)
	case err == nil:
		err = json.Unmarshal(buf, &conf)
		if err != nil {
			return err
		}
		setConfigDefaults(&conf)
	default:
		return err
	}

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
		"serve and listen only through HTTPS. Requires -ssl-cert and "+
			"-ssl-key to be set",
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
	flag.Usage = printUsage

	// Parse command line arguments
	flag.Parse()
	if cache.Size < 0 {
		return errors.New("cache size must be a positive number")
	}
	arg := flag.Arg(0)
	if arg == "" {
		arg = "debug"
	}

	// Can't daemonise in windows, so only args they have is "start" and "help"
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
		gspt.SetProcTitle(os.Args[0])
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
