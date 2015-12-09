// Package tmpl compiles HTML templates to be used for rendering pages during
// dynamic content insertion
package tmpl

import (
	"encoding/json"
	"io/ioutil"
	"meguca/config"
)

type templateStore struct {
	parts []string
	hash  string
}

type templateMap map[string]map[string]templateStore

// Resources exports temolates and their hashes by language
var Resources templateMap

// Compile reads template HTML from disk, injext dynamic variables, hashes and
// exports them
func Compile() error {
	// Only one for now, but there will be more later
	raw := map[string]string{}
	for _, name := range []string{"index"} {
		file, err := ioutil.ReadFile("./tmpl/html/" + name + ".html")
		if err != nil {
			return err
		}
		raw[name] = string(file)
	}
	if err := indexTemplate(raw["index"]); err != nil {
		return err
	}
	return nil
}

// indexTemplate compiles the HTML template for thread and board pages of the
// imageboard
func indexTemplate(tmpl string) error {
	vars := map[string]string{}
	if js, err := json.Marshal(config.ClientConfig); err != nil {
		return err
	} else {
		vars["config"] = string(js)
	}
	return nil
}
