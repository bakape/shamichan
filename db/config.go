// Configuration loading, reloading and setting

package db

import "github.com/bakape/meguca/config"

// TODO: Actual configuration loading

func loadConfigs() error {
	return config.Set(config.Defaults)
}
