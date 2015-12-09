package main

import (
	"github.com/go-errors/errors"
	"log"
	"meguca/config"
	"meguca/lang"
	"meguca/tmpl"
)

func main() {
	if err := config.Load(); err != nil {
		log.Fatalf("Error loading config files: %#v\n", err)
	}
	if err := lang.Load(); err != nil {
		log.Fatalf("Error loading language packs: %#v\n", err)
	}
	logError(tmpl.Compile, "Error compiling templates")
}

func logError(routine func() error, msg string) {
	if err := routine(); err != nil {
		log.Fatalf(msg+":\n%v", err.(*errors.Error).ErrorStack())
	}
}
