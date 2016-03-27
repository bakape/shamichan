// Package server handles client requests, both for HTML page rendering and
// websocket connections.
package server

import (
	"fmt"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/templates"
	"log"
	"os"
	"runtime"
)

var (
	// debugMode denotes the server has been started with the `debug` parameter.
	// This will cause not to spawn a daemon and stay attached to the launching
	// shell.
	debugMode bool
	isWindows = runtime.GOOS == "windows"

	// Is assigned in ./daemon.go to control/spawn a daemon process. That file
	// is never compiled on Windows and this function is never called.
	handleDaemon func(string)
)

// Start parses command line arguments and initializes the server.
func Start() {
	// Parse command line arguments
	if len(os.Args) < 2 {
		printUsage()
	}
	arg := os.Args[1]

	// Can't daemonise in windows, so only args they have is "start" and "help"
	if isWindows {
		switch arg {
		case "debug":
			fallthrough
		case "start":
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

var arguments = map[string]string{
	"start":   "start the meguca server",
	"stop":    "stop a running daemonised meguca server",
	"restart": "combination of stop + start",
	"debug":   "start server in debug mode without deamonising",
	"help":    "print this help text",
}

// Constructs and prints the CLI help text
func printUsage() {
	usage := "usage: meguca "
	var help string
	toPrint := []string{"start"}
	if !isWindows {
		toPrint = append(toPrint, []string{"stop", "restart"}...)
	} else {
		arguments["debug"] = `alias of "start"`
	}
	toPrint = append(toPrint, []string{"debug", "help"}...)
	for i, arg := range toPrint {
		if i != 0 {
			usage += "|"
		}
		usage += arg
		help += fmt.Sprintf("  %s\t  %s\n", arg, arguments[arg])
	}
	os.Stderr.WriteString(usage + "\n" + help)
	os.Exit(1)
}

func startServer() {
	fns := []func() error{
		config.LoadConfig,
		templates.Compile,
		db.LoadDB,
		startWebServer,
	}
	for _, fn := range fns {
		err := fn()
		if err != nil {
			log.Fatal(err)
		}
	}
}
