package main

import (
	"os/user"
)

func main() {
	current, err := user.Current()
	throw(err)
	if current.Uid == "0" {
		panic("Refusing to  run as root")
	}
	loadConfig()
	loadLanguagePacks()
	compileTemplates()
	loadDB()
}
