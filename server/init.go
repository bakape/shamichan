// Package server handles client requests, both for HTML page rendering and
// websocket connections.
package server

import (
	"fmt"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/templates"
	"github.com/bakape/meguca/util"
	"github.com/sevlyar/go-daemon"
	"log"
	"os"
	"os/user"
	"strconv"
	"syscall"
	"time"
)

// debugMode denotes the server has been started with the `debug` parameter.
// This will cause it to assume, it is run from the project source root
// directory. Also changes some folder paths to be located under non-system
// paths like `/etc` and `/var`.
var debugMode bool

// Start parses command line arguments and initializes the server.
func Start() {
	// Parse command line arguments
	if len(os.Args) < 2 {
		printUsage()
	}
	arg := os.Args[1]
	switch arg {
	case "init": // For internal use only
		os.Exit(0)
	case "debug":
		debugMode = true
	case "start":
	case "stop":
		killDaemon()
		os.Exit(0)
	case "restart":
		killDaemon()
	default:
		printUsage()
	}

	config.LoadConfig(debugMode)

	if !debugMode {
		daemonise()
	} else {
		webRoot = "./www"
		imageWebRoot = "./img"
		createDebugDirs()
		startServer()
	}
}

func printUsage() {
	fmt.Print(`usage: meguca.v2 [ start | stop | restart | debug | help ]
    start   - start the meguca server
    stop    - stop a running daemonised meguca server
    restart - combination of stop + start
    debug   - debug mode
    help    - print this help text
`)
	os.Exit(1)
}

func startServer() {
	templates.Compile()
	db.LoadDB()
	startWebServer()
}

// Configuration variables for handling daemons
var daemonContext = &daemon.Context{
	PidFileName: "/var/run/meguca.pid",
	LogFileName: "/var/log/meguca.error.log",
	Credential:  getCredentials(),
}

func getCredentials() *syscall.Credential {
	us, err := user.Lookup("meguca")
	util.Throw(err)
	uid, err := strconv.Atoi(us.Gid)
	util.Throw(err)
	gid, err := strconv.Atoi(us.Gid)
	util.Throw(err)
	return &syscall.Credential{
		Uid: uint32(uid),
		Gid: uint32(gid),
	}
}

// Spawn a detached process to work in the background
func daemonise() {
	child, err := daemonContext.Reborn()
	if err != nil {
		cur, err := user.Current()
		util.Throw(err)
		fmt.Print(cur.Uid)
		if cur.Uid != "0" {
			log.Fatalln("Must be started as root, if in non-debug mode")
		}
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
	util.Throw(daemon.ServeSignals())
	log.Println("Server terminated")
}

// Terminate the running meguca server daemon
func killDaemon() {
	proc, err := daemonContext.Search()
	if err != nil && (!os.IsNotExist(err) && err.Error() != "EOF") {
		panic(err)
	}
	if proc != nil {
		util.Throw(proc.Signal(syscall.SIGTERM))

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

// Create all needed directories for server operation in debug mode
func createDebugDirs() {
	for _, dir := range [...]string{"src", "thumb", "mid"} {
		util.Throw(os.MkdirAll("./img/"+dir, 0750))
	}
}
