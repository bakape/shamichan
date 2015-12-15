// Package lang parses JSON language packs and exports them
package lang

import (
	"encoding/json"
	"io/ioutil"
	"meguca/config"
	. "meguca/util"
)

type languagePack struct {
	Imager map[string]string
}
type packMap map[string]languagePack

// Langs contains languagepack structs for each language
var Langs packMap

// Load reads and exports server-side language packs from JSON
func Load() {
	Langs = packMap{}
	for _, lang := range config.Config.Lang.Enabled {
		file, err := ioutil.ReadFile("./lang/" + lang + "/server.json")
		Throw(err)
		var parsed languagePack
		Throw(json.Unmarshal(file, &parsed))
		Langs[lang] = parsed
	}
}
