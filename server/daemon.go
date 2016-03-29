// +build linux darwin

// Daemonisation logic for the server

package server

import (
	"github.com/sevlyar/go-daemon"
	"log"
	"os"
	"syscall"
	"time"
)

func init() {
	handleDaemon = func(arg string) {
		switch arg {
		case "debug":
			startServer()
		case "stop":
			killDaemon()
			fallthrough
		case "init": // For internal use only
			os.Exit(0)
		case "restart":
			killDaemon()
			fallthrough
		case "start":
			daemonise()
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
	if err != nil && err.Error() == "resource temporarily unavailable" {
		log.Fatalln("Error: Server already running")
	}
	if child != nil {
		return
	}
	daemonised = true
	defer daemonContext.Release()
	log.Println("Server started ------------------------------------")
	go startServer()
	if err := daemon.ServeSignals(); err != nil {
		log.Fatalf("Daemon runtime error: %s\n", err)
	}
	log.Println("Server terminated")
}

// Terminate the running meguca server daemon
func killDaemon() {
	proc, err := daemonContext.Search()
	if err != nil && (!os.IsNotExist(err) && err.Error() != "EOF") {
		log.Fatalf("Error locating running daemon: %s\n", err)
	}
	if proc != nil {
		if err := proc.Signal(syscall.SIGTERM); err != nil {
			log.Fatalf("Error killing running daemon: %s\n", err)
		}

		// Assertain process has exited
		for {
			if err := proc.Signal(syscall.Signal(0)); err != nil {
				if err.Error() == "os: process already finished" {
					break
				}
				log.Fatalf("Error ascertaining daemon exited: %s\n", err)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}
