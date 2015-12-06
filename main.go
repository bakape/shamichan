package main

import (
	"log"
	"meguca/config"
)

func main() {
	err := config.Load()
	if err != nil {
		log.Fatalf("Error loading config files: %#v\n", err)
	}
}
