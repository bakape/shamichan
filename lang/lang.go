// Package lang parses JSON language packs and exports them
package lang

import (
	"encoding/json"
	"io/ioutil"
	"meguca/config"
)

// Struct maps server-side JSON languagepacks to Go types
type Struct struct {
	ShowSeconds, WorksBestWith string
	Imager, Tmpl               map[string]string
	Opts                       struct {
		Tabs   []string
		Modes  map[string]string
		Labels map[string][2]string
	}
}

// Map contains a map of all languages to their languagepacks
type Map map[string]Struct

// Langs contains languagepack structd for each language
var Langs Map

// Load reads and exports server-side language packs from JSON
func Load() (err error) {
	Langs = Map{}
	for _, lang := range config.Config.Lang.Enabled {
		var file []byte
		file, err = ioutil.ReadFile("./lang/" + lang + "/server.json")
		if err != nil {
			return
		}
		parsed := Struct{}
		if err = json.Unmarshal(file, &parsed); err != nil {
			return
		}
		Langs[lang] = parsed
	}
	return
}
