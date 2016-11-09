package lang

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
)

var (
	// LangDir is the path to the language pack directory. Overrideable
	// for tests.
	Dir = "lang"

	// Packs contains all loaded language packs
	Packs map[string]Pack
)

// Pack contains a localization language pack for a single language
type Pack struct{}

// Load loads and parses all JSON language packs
func Load() error {
	dirs, err := ioutil.ReadDir(Dir)
	if err != nil {
		return err
	}

	packs := make(map[string]Pack, len(dirs)-2)
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}

		lang := d.Name()
		buf, err := ioutil.ReadFile(filepath.Join(Dir, lang, "server.json"))
		if err != nil {
			return err
		}
		var p Pack
		if err := json.Unmarshal(buf, &p); err != nil {
			return err
		}
		packs[lang] = p
	}

	return nil
}
