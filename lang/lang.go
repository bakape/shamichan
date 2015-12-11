// Package lang parses JSON language packs and exports them
package lang

import (
	"encoding/json"
	"io/ioutil"
	"meguca/config"
	. "meguca/util"
)

// Struct maps server-side JSON languagepacks to Go types
type Struct struct {
	Imager map[string]string
}

// Map contains a map of all languages to their languagepacks
type Map map[string]Struct

// Langs contains languagepack structd for each language
var Langs Map

// Load reads and exports server-side language packs from JSON
func Load() {
	Langs = Map{}
	for _, lang := range config.Config.Lang.Enabled {
		file, err := ioutil.ReadFile("./lang/" + lang + "/server.json")
		Throw(err)
		parsed := Struct{}
		Throw(json.Unmarshal(file, &parsed))
		Langs[lang] = parsed
	}
}
