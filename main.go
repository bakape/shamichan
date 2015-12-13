package main

import (
	"meguca/config"
	"meguca/lang"
	"meguca/server"
	"meguca/tmpl"
)

func main() {
	config.Load()
	lang.Load()
	tmpl.Compile()
	server.Load()
}
