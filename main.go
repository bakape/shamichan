package main

import (
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
	if err := tmpl.Compile(); err != nil {
		log.Fatalf("Error compiling templates: %#v\n", err)
	}
}
