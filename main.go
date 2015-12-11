package main

import (
	"meguca/config"
	"meguca/db"
	"meguca/lang"
	"meguca/tmpl"
)

func main() {
	config.Load()
	lang.Load()
	tmpl.Compile()
	db.Load()
}
