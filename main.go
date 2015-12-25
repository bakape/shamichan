package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"strconv"
)

func main() {
	current, err := user.Current()
	throw(err)
	if current.Uid == "0" {
		panic("Refusing to  run as root")
	}
	loadConfig()
	pidFile()
	loadLanguagePacks()
	compileTemplates()
	loadDB()
	startServer()
}

// Handle any previous meguca processes and write down the cuurent PID to a file
func pidFile() {
	const path = "./.pid"

	// Read old PID file
	if buf, err := ioutil.ReadFile(path); err == nil {
		// Kill previous meguca instance, if any
		if !config.Hard.Debug {
			pid, err := strconv.Atoi(string(buf))
			throw(err)
			process, err := os.FindProcess(pid)
			throw(err)
			if err := process.Kill(); err == nil {
				log.Printf("Killed already running instance with PID %v\n", pid)
			} else if err.Error() != "os: process already finished" {
				panic(err)
			}
		}

		throw(os.Remove(path))
	} else if !os.IsNotExist(err) {
		panic(err)
	}

	// Write new PID file
	throw(ioutil.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0660))
}
