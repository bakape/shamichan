// Package lang parses JSON language packs and exports them
package lang

import (
	"encoding/json"
	"github.com/go-errors/errors"
	"io/ioutil"
	"meguca/config"
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
func Load() error {
	Langs = Map{}
	for _, lang := range config.Config.Lang.Enabled {
		file, err := ioutil.ReadFile("./lang/" + lang + "/server.json")
		if err != nil {
			return errors.Wrap(err, 0)
		}
		parsed := Struct{}
		if err := json.Unmarshal(file, &parsed); err != nil {
			return errors.Wrap(err, 0)
		}
		Langs[lang] = parsed
	}
	return nil
}
