// +build linux darwin

// Daemonization logic for the server

package server

import (
	"os"
	"syscall"
	"time"

	"github.com/go-playground/log"
	"github.com/sevlyar/go-daemon"

	"meguca/assets"
	"meguca/config"
	"meguca/log"
)

func init() {
	handleDaemon = func(arg string) {
		if config.ImagerMode == config.ImagerOnly {
			daemonContext.PidFileName = ".imager.pid"
			daemonContext.LogFileName = "imager_error.log"
		}

		switch arg {
		case "debug":
			mlog.Init(mlog.Console)
			mlog.ConsoleHandler.SetDisplayColor(true)
			go assets.WatchVideoDir()
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
			mlog.Init(mlog.Console)
			mlog.ConsoleHandler.SetDisplayColor(false)
			go assets.WatchVideoDir()
			daemonize()
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
func daemonize() {
	child, err := daemonContext.Reborn()
	if err != nil && err.Error() == "resource temporarily unavailable" {
		log.Fatal("Error: Server already running")
	}
	if child != nil {
		return
	}
	daemonized = true
	defer daemonContext.Release()
	log.Info("Server started ------------------------------------")

	go startServer()
	if err := daemon.ServeSignals(); err != nil {
		log.Fatalf("daemon runtime error: %s\n", err)
	}
	log.Info("server terminated")
}

// Terminate the running meguca server daemon
func killDaemon() {
	proc := findDaemon()
	if proc != nil {
		if err := proc.Signal(syscall.SIGTERM); err != nil {
			log.Fatalf("error killing running daemon: %s\n", err)
		}

		// Ascertain process has exited
		for {
			if err := proc.Signal(syscall.Signal(0)); err != nil {
				if err.Error() == "os: process already finished" {
					break
				}
				log.Fatalf("error ascertaining daemon exited: %s\n", err)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// Find the running daemonized meguca server process
func findDaemon() *os.Process {
	proc, err := daemonContext.Search()
	if err != nil && (!os.IsNotExist(err) && err.Error() != "EOF") {
		log.Fatalf("error locating running daemon: %s\n", err)
	}
	return proc
}
