package config

// Configurations of this specific instance passed from config file.
// Immutable after loading.
var Server ServerConfigs

// Configurations of this specific instance passed from config file
type ServerConfigs struct {
	/// Indicates this server is behind a reverse proxy and can honour
	/// X-Forwarded-For and similar headers
	ReverseProxied bool `short:"r" long:"reverse-proxied" description:"Indicates this server is behind a reverse proxy and can honour X-Forwarded-For and similar headers"`

	// Database address to connect to
	Database string `short:"d" long:"database" description:"Database address to connect to" default:"postgres://meguca:meguca@localhost:5432/meguca"`

	// Address for the server to listen on
	Address string `short:"a" long:"address" description:"Address for the server to listen on" default:"127.0.0.1:8001"`
}
