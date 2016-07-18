// Package server handles client requests, both for HTML page rendering and
// websocket connections.
package server

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/imager"
	"github.com/bakape/meguca/templates"
)

var (
	// debugMode denotes the server has been started with the `debug` parameter.
	// This will cause not to spawn a daemon and stay attached to the launching
	// shell.
	debugMode  bool
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
		"debug":   "start server in debug mode without deamonising (default)",
		"help":    "print this help text",
	}
)

// Start parses command line arguments and initializes the server.
func Start() {
	// Define flags
	flag.StringVar(
		&address,
		"http-addr",
		":8000",
		"address to listen on for incomming HTTP connections",
	)
	flag.StringVar(
		&db.Address,
		"db-addr",
		"localhost:28015",
		"address of the RethinkDB server to connect to",
	)
	flag.StringVar(
		&db.DBName,
		"db-name",
		"meguca",
		"name of the RethinkDB database to use",
	)
	flag.BoolVar(
		&ssl,
		"ssl",
		false,
		"serve and listen only through HTTPS. Requires -ssl-cert and "+
			"-ssl-key to be set",
	)
	flag.StringVar(&sslCert, "ssl-cert", "", "path to SSL certificate")
	flag.StringVar(
		&config.AllowedOrigin,
		"origin",
		"localhost:8000",
		"outward origin of the server. Must match location.host in the browser.",
	)
	flag.BoolVar(
		&auth.IsReverseProxied,
		"reverse-proxied",
		false,
		"assume server is behind reverse proxy, when resolving client IPs",
	)
	flag.StringVar(
		&auth.ReverseProxyIP,
		"reverse-proxy-IP",
		"",
		"IP of the reverse proxy. Only needed, when reverse proxy is not on localhost.",
	)
	flag.BoolVar(&enableGzip, "gzip", false, "compress all traffic with gzip")
	flag.Usage = printUsage

	// Parse command line arguments
	flag.Parse()
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
	fns := []func() error{
		db.LoadDB,
		templates.Compile,
		imager.InitImager,
		startWebServer,
	}
	for _, fn := range fns {
		err := fn()
		if err != nil {
			log.Fatal(err)
		}
	}
}
