//go:generate go-bindata -o bin_data.go --pkg lang --nometadata --prefix ../../../../lang ../../../../lang/...

package lang

import (
	"encoding/json"
	"meguca/config"
	"path/filepath"
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
	Forms           map[string][2]string
	UI, Options     map[string]string
	Templates       map[string][]string
	Common          struct {
		Posts   map[string]string    `json:"posts"`
		Plurals map[string][2]string `json:"plurals"`
		Forms   map[string][2]string `json:"forms"`
		Time    map[string][]string  `json:"time"`
		UI      map[string]string    `json:"ui"`
		Sync    []string             `json:"sync"`
	}
}

// Loads and parses the selected JSON language pack
func Load() (err error) {
	lang := config.Get().DefaultLang

	buf, err := Asset(filepath.Join(lang, "server.json"))
	if err != nil {
		return
	}
	err = json.Unmarshal(buf, &pack)
	if err != nil {
		return
	}

	buf, err = Asset(filepath.Join(lang, "common.json"))
	if err != nil {
		return
	}
	err = json.Unmarshal(buf, &pack.Common)
	if err != nil {
		return
	}
	pack.ID = lang

	return
}

// Returns the loaded language pack
func Get() Pack {
	return pack
}
