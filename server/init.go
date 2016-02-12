// Package server handles client requests, both for HTML page rendering and
// websocket connections.
package server

import (
	"fmt"
	"github.com/sevlyar/go-daemon"
	"log"
	"os"
	"syscall"
	"time"
)

// Start parses command line arguments and initializes the server.
func Start() {
	chdirToSource()
	loadConfig()
	createDirs()

	// Parse command line arguments
	if len(os.Args) < 2 {
		printUsage()
	}
	arg := os.Args[1]
	switch arg {
	case "init": // For internal use only
		os.Exit(0)
	case "debug":
		config.Debug = true
	case "start":
	case "stop":
		killDaemon()
		os.Exit(0)
	case "restart":
		killDaemon()
	default:
		printUsage()
	}

	if !config.Debug {
		daemonise()
	} else {
		startServer()
	}
}

// Changes the working directory to meguca's source directory, so we can
// properly read configs, serve web resources, etc., while still being able to
// run from meguca from any directory
// TODO: CL flags to override this
func chdirToSource() {
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		workDir := gopath + "/src/github.com/bakape/meguca"
		throw(os.Chdir(workDir))
		daemonContext.WorkDir = workDir
	}
}

func printUsage() {
	fmt.Print(`usage: meguca.v2 [ start | stop | debug | help ]
    start   - start the meguca server
    stop    - stop a running daemonised meguca server
    restart - combination of stop + start
    debug   - debug mode
    help    - print this help text
`)
	os.Exit(1)
}

func startServer() {
	loadLanguagePacks()
	compileTemplates()
	loadDB()
	startWebServer()
}

// Configuration variables for handling daemons
var daemonContext = &daemon.Context{
	PidFileName: "./.pid",
	LogFileName: "./error.log",
}

// Spawn a detached process to work in the background
func daemonise() {
	child, err := daemonContext.Reborn()
	if err != nil {
		if err.Error() == "resource temporarily unavailable" {
			fmt.Println("Error: Server already running")
			os.Exit(1)
		}
		panic(err)
	}
	if child != nil {
		return
	}
	defer daemonContext.Release()
	log.Println("Server started ------------------------------------")
	go startServer()
	throw(daemon.ServeSignals())
	log.Println("Server terminated")
}

// Terminate the running meguca server daemon
func killDaemon() {
	proc, err := daemonContext.Search()
	if err != nil && (!os.IsNotExist(err) && err.Error() != "EOF") {
		panic(err)
	}
	if proc != nil {
		throw(proc.Signal(syscall.SIGTERM))

		// Assertain process has exited
		for {
			if err := proc.Signal(syscall.Signal(0)); err != nil {
				if err.Error() == "os: process already finished" {
					break
				}
				panic(err)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// Create all needed directories for server operation
func createDirs() {
	for _, dir := range [...]string{"src", "thumb", "mid"} {
		throw(os.MkdirAll("./img/"+dir, 0750))
	}
}
