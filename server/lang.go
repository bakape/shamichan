/*
 Parses JSON language packs and exports them
*/

package server

import (
	"io/ioutil"
)

// LanguagePack contains the server-side string mappings of a single language
type LanguagePack struct {
	Imager map[string]string
}
type packMap map[string]LanguagePack

// langs contains languagepack structs for each language
var langs packMap

// loadLanguagePacks reads and exports server-side language packs from JSON
func loadLanguagePacks() {
	langs = packMap{}
	for _, lang := range config.Lang.Enabled {
		file, err := ioutil.ReadFile("./lang/" + lang + "/server.json")
		throw(err)
		var parsed LanguagePack
		unmarshalJSON(file, &parsed)
		langs[lang] = parsed
	}
}
