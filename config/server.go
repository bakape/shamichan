package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Configurations of this specific instance passed from config file.
// Immutable after loading.
var Server ServerConfigs

// ImagerModeType is the imager functionality setting for this meguca process
type ImagerModeType int

const (
	// IntegratedImager is regular and imager functionality both handled by this process
	IntegratedImager ImagerModeType = iota

	// NoImager is imager functionality not handled by this process
	NoImager

	// ImagerOnly is only imager functionality handled by this process
	ImagerOnly
)

// Configurations of this specific instance passed from config file
type ServerConfigs struct {
	Debug     bool
	Database  string
	CacheSize float64 `json:"cache_size"`
	Server    struct {
		ReverseProxied bool `json:"reverse_proxied"`
		Address        string
	}
	Test struct {
		Database string
	}
}

// Load configs from JSON or defaults, if none present
func (c *ServerConfigs) Load() (err error) {
	path := "config.json"
	prefix := ""
try:
	f, err := os.Open(filepath.Join(prefix, path))
	if err != nil {
		if os.IsNotExist(err) {
			_, err = os.Stat(filepath.Join(prefix, "go.mod"))
			switch {
			case err == nil:
				// Reached the root dir
				c.setDefaults()
				return
			case os.IsNotExist(err):
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
	c.Debug = true
	c.Database = "postgres://meguca:meguca@localhost:5432/meguca?sslmode=disable"
	c.CacheSize = 128
	c.Server.Address = ":8000"
	c.Test.Database = "postgres://meguca:meguca@localhost:5432/meguca_test?sslmode=disable"
}
