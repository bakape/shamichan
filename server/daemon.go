// +build linux darwin

// Daemonisation logic for the server

package server

import (
	"github.com/sevlyar/go-daemon"
)

func init() {
	handleDaemon = func(arg string) {
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
	}
}

// Configuration variables for handling daemons
var daemonContext = &daemon.Context{
	PidFileName: ".pid",
	LogFileName: "error.log",
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
