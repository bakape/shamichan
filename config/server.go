package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Configurations of this specific instance passed from config file.
// Immutable after loading.
var Server ServerConfigs

// Configurations of this specific instance passed from config file
type ServerConfigs struct {
	Database  string
	CacheSize float64 `json:"cache_size"`
	Server    struct {
		ReverseProxied bool `json:"reverse_proxied"`
		Address        string
	}
}

// Load configs from JSON or defaults, if none present
func (c *ServerConfigs) Load() (err error) {
	c.setDefaults()

	var (
		prefix, abs string
		path        = "config.json"
	)
try:
	f, err := os.Open(filepath.Join(prefix, path))
	if err != nil {
		if os.IsNotExist(err) {
			_, err = os.Stat(filepath.Join(prefix, "go.mod"))
			switch {
			case err == nil:
				return // Reached the project root dir
			case os.IsNotExist(err):
				if prefix != "" {
					abs, err = filepath.Abs(prefix)
					if err != nil {
						return
					}
					if abs == "/" {
						return // Reached the system root dir
					}
				}
				// Go up one dir
				prefix = filepath.Join("..", prefix)
				goto try
			default:
				return
			}

		}
		return
	}
	defer func() {
		if f != nil {
			f.Close()
		}
	}()

	return json.NewDecoder(f).Decode(c)
}

func (c *ServerConfigs) setDefaults() {
	c.Database = "postgres://meguca:meguca@localhost:5432/meguca?sslmode=disable"
	c.CacheSize = 128
	c.Server.Address = ":8000"
}
