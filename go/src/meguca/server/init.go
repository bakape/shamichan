// Package server handles client requests for HTML page rendering, JSON and
// websocket connections
package server

import (
	"bytes"
	"flag"
	"fmt"
	"log"
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
)

var (
	// debugMode denotes the server has been started with the `debug` parameter.
	// This will cause not to spawn a daemon and stay attached to the launching
	// shell.
	daemonised bool
	isWindows  = runtime.GOOS == "windows"

	// Is assigned in ./daemon.go to control/spawn a daemon process. That file
	// is never compiled on Windows and this function is never called.
	handleDaemon func(string)

	// CLI mode arguments and descriptions
	arguments = map[string]string{
		"start":   "start the meguca server",
		"stop":    "stop a running daemonised meguca server",
		"restart": "combination of stop + start",
		"debug":   "start server in debug mode without daemonizing (default)",
		"help":    "print this help text",
	}
)

// Start parses command line arguments and initializes the server.
func Start() {
	// Define flags
	flag.StringVar(
		&address,
		"a",
		":8000",
		"address to listen on for incoming HTTP connections",
	)
	flag.Float64Var(&cache.Size, "c", 1<<7, "cache size in MB")
	flag.StringVar(
		&db.ConnArgs,
		"d",
		`user=meguca password=meguca dbname=meguca sslmode=disable`,
		"PostgreSQL connection arguments",
	)
	flag.BoolVar(
		&ssl,
		"s",
		false,
		"serve and listen only through HTTPS. Requires -ssl-cert and "+
			"-ssl-key to be set",
	)
	flag.StringVar(&sslCert, "S", "", "path to SSL certificate")
	flag.BoolVar(
		&auth.IsReverseProxied,
		"r",
		false,
		"assume server is behind reverse proxy, when resolving client IPs",
	)
	flag.StringVar(
		&auth.ReverseProxyIP,
		"R",
		"",
		"IP of the reverse proxy. Only needed, when reverse proxy is not on localhost.",
	)
	flag.BoolVar(&enableGzip, "g", false, "compress all traffic with gzip")
	flag.Usage = printUsage

	// Parse command line arguments
	flag.Parse()
	if cache.Size < 0 {
		log.Fatal("cache size must be a positive number")
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
		handleDaemon(arg)
	}
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
