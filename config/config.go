// Package config parses JSON configuration files and exports the configuration
// for server-side use and the public availability JSON struct, which includes
// a small subset of the server configuration.
package config

import (
	"sync"
	"time"
)

var (
	// Ensures no reads happen, while the configuration is reloading
	globalMu, boardConfigMu, boardsMu sync.RWMutex

	// Contains currently loaded server configuration. Intialised to an empty
	// struct pointer, so we don't have to explicitly initialise it in all our
	// tests.
	global = &Configs{}

	// All existing boards
	boards []string

	// Map of board IDs to their cofiguration structs
	boardConfigs map[string]BoardConfigs

	// JSON of client-accessable configuration
	clientJSON []byte

	// Hash of the gloabal configs. Used for live reloading configuration on the
	// client
	hash string
)

// Configs stores the global configuration
type Configs struct {
	SSL              bool          `json:"-"`
	TrustProxies     bool          `json:"-" gorethink:"trustProxies"`
	Gzip             bool          `json:"-" gorethink:"gzip"`
	Prune            bool          `json:"-" gorethink:"prune"`
	Radio            bool          `json:"radio" gorethink:"radio"`
	WebmAudio        bool          `json:"-" gorethink:"webmAudio"`
	Hats             bool          `json:"hats" gorethink:"hats"`
	MaxWidth         uint16        `json:"-" gorethink:"maxWidth"`
	MaxHeight        uint16        `json:"-" gorethink:"maxHeight"`
	MaxThreads       int           `json:"-" gorethink:"maxThreads"`
	MaxBump          int           `json:"-" gorethink:"maxBump"`
	JPEGQuality      int           `json:"-"`
	PNGQuality       int           `json:"-"`
	ThreadCooldown   int           `json:"threadCooldown" gorethink:"threadCooldown"`
	MaxSubjectLength int           `json:"maxSubjectLength" gorethink:"maxSubjectLength"`
	MaxSize          int64         `json:"-" gorethink:"maxSize"`
	Origin           string        `json:"-" gorethink:"origin"`
	SSLCert          string        `json:"-"`
	SSLKey           string        `json:"-"`
	Frontpage        string        `json:"-" gorethink:"frontpage"`
	DefaultCSS       string        `json:"defaultCSS" gorethink:"defaultCSS"`
	Salt             string        `json:"-" gorethink:"salt"`
	ExcludeRegex     string        `json:"excludeRegex" gorethink:"excludeRegex"`
	FeedbackEmail    string        `json:"-" gorethink:"feedbackEmail"`
	Links            [][2]string   `json:"links" gorethink:"links"`
	Spoilers         []uint8       `json:"spoilers" gorethink:"spoliers"`
	SessionExpiry    time.Duration `json:"-" gorethink:"sessionExpiry"`
}

// Defaults contains the default server configuration values
var Defaults = Configs{
	Origin:           "localhost:8000",
	TrustProxies:     false,
	Gzip:             true,
	SSL:              false,
	SSLCert:          "",
	SSLKey:           "",
	Frontpage:        "",
	MaxThreads:       100,
	MaxBump:          1000,
	Links:            [][2]string{{"4chan", "http://www.4chan.org/"}},
	Prune:            true,
	WebmAudio:        true,
	Hats:             false,
	JPEGQuality:      90,
	PNGQuality:       20,
	MaxSize:          3145728,
	MaxHeight:        6000,
	MaxWidth:         6000,
	Spoilers:         []uint8{0},
	DefaultCSS:       "moe",
	ThreadCooldown:   60,
	MaxSubjectLength: 50,
	Salt:             "LALALALALALALALALALALALALALALALALALALALA",
	ExcludeRegex:     "/[\u2000-\u200f\u202a-\u202f\u205f-\u206f]+/g",
	Radio:            false,
	SessionExpiry:    30,
	FeedbackEmail:    "admin@email.com",
}

// BoardConfigs stores overall board configuration
type BoardConfigs struct{}

// Get returns a pointer to the current server configuration struct. Callers
// should not modify this struct.
func Get() *Configs {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return global
}

// Set sets the internal configuration struct. To be used only in tests.
func Set(c Configs) {
	globalMu.Lock()
	defer globalMu.Unlock()
	global = &c
}

// GetClient returns punlic availability configuration JSON and a truncated
// configuration MD5 hash
func GetClient() ([]byte, string) {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return clientJSON, hash
}

// SetClient sets the client configuration JSON and hash. To be used only in
// tests.
func SetClient(json []byte, cHash string) {
	globalMu.Lock()
	defer globalMu.Unlock()
	clientJSON = json
	hash = cHash
}

// GetBoards returns all boards currently existing. Receivers must not modify
// the slice.
func GetBoards() []string {
	boardsMu.RLock()
	defer boardsMu.RUnlock()
	return boards
}

// SetBoards updates the all boards currently existing set
func SetBoards(b []string) {
	boardsMu.Lock()
	defer boardsMu.Unlock()
	boards = b
}
