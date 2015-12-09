package main

import (
	"github.com/go-errors/errors"
	"log"
	"meguca/config"
	"meguca/lang"
	"meguca/tmpl"
)

func main() {
	logError(config.Load, "Error loading config files")
	logError(lang.Load, "Error loading language packs")
	logError(tmpl.Compile, "Error compiling templates")
}

func logError(routine func() error, msg string) {
	if err := routine(); err != nil {
		log.Fatalf(msg+":\n%v", err.(*errors.Error).ErrorStack())
	}
}
