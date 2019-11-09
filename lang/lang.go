package lang

import (
	"encoding/json"
	"fmt"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/static"
)

var (
	// Currently used language pack
	pack Pack

	// Precompiled table of relations between browser Accept-Language HTTP
	// header values and internal POSIX language codes
	languageCodes map[string]string
)

// Pack contains a localization language pack for a single language
type Pack struct {
	ID              string
	Tabs, SortModes []string
	UI, Options     map[string]string
	Forms           map[string][2]string
	Templates       map[string][]string
	Common          struct {
		UI      map[string]string    `json:"ui"`
		Format  map[string]string    `json:"format"`
		Posts   map[string]string    `json:"posts"`
		Plurals map[string][2]string `json:"plurals"`
		Forms   map[string][2]string `json:"forms"`
		Time    map[string][]string  `json:"time"`
		Sync    []string             `json:"sync"`
	}
}

// Load loads and parses the selected JSON language pack
func Load() (err error) {
	lang := config.Get().DefaultLang

	readJSON := func(file string, dst interface{}) (err error) {
		f, err := static.FS.Open(fmt.Sprintf("/lang/%s/%s", lang, file))
		if err != nil {
			return
		}
		defer f.Close()
		return json.NewDecoder(f).Decode(dst)
	}

	err = readJSON("server.json", &pack)
	if err != nil {
		return
	}
	err = readJSON("common.json", &pack.Common)
	if err != nil {
		return
	}

	pack.ID = lang

	return
}

// Get returns the loaded language pack
func Get() Pack {
	return pack
}
