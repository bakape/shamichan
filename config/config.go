// Package config parses JSON configuration files and exports the configuration
// for server-side use and the public availability JSON struct, which includes
// a small subset of the server configuration.
package config

import (
	"encoding/json"
	"strconv"
	"sync"
	"time"

	"github.com/bakape/meguca/util"
)

var (
	// Ensures no reads happen, while the configuration is reloading
	globalMu, boardsMu sync.RWMutex

	// Contains currently loaded global server configuration
	global *Configs

	// Map of board IDs to their cofiguration structs
	boardConfigs map[string]BoardConfigs

	// JSON of client-accessable configuration
	clientJSON []byte

	// Hash of the gloabal configs. Used for live reloading configuration on the
	// client.
	hash string
)

// Default string for the FAQ panel
const defaultFAQ = `Upload size limit is 3 MB
Accepted upload file types: JPG, JPEG, PNG, GIF, WEBM, SVG, PDF, MP3, MP4, OGG
<hr>Hash commands:
#d100 #2d100 - Roll dice
#flip - Coin flip
#8ball - An 8ball
#queue - Print r/a/dio song queue
#sw24:15 #sw2:24:15 #sw24:15+30 #sw24:15-30 - Syncronised duration timer`

// Configs stores the global configuration
type Configs struct {
	Prune            bool   `json:"-" gorethink:"prune"`
	Radio            bool   `json:"radio" gorethink:"radio"`
	WebmAudio        bool   `json:"-" gorethink:"webmAudio"`
	Hats             bool   `json:"hats" gorethink:"hats"`
	MaxWidth         uint16 `json:"-" gorethink:"maxWidth"`
	MaxHeight        uint16 `json:"-" gorethink:"maxHeight"`
	MaxThreads       int    `json:"-" gorethink:"maxThreads"`
	MaxBump          int    `json:"-" gorethink:"maxBump"`
	JPEGQuality      int    `json:"-"`
	PNGQuality       int    `json:"-"`
	ThreadCooldown   int    `json:"threadCooldown" gorethink:"threadCooldown"`
	MaxSubjectLength int    `json:"maxSubjectLength" gorethink:"maxSubjectLength"`
	MaxSize          int64  `json:"-" gorethink:"maxSize"`
	DefaultLang      string `json:"defaultLang" gorethink:"defaultLang"`
	Frontpage        string `json:"-" gorethink:"frontpage"`
	Origin           string `json:"-" gorethink:"origin"`
	DefaultCSS       string `json:"defaultCSS" gorethink:"defaultCSS"`
	Salt             string `json:"-" gorethink:"salt"`
	ExcludeRegex     string `json:"-" gorethink:"excludeRegex"`
	FeedbackEmail    string `json:"-" gorethink:"feedbackEmail"`
	FAQ              string
	Boards           []string      `json:"boards" gorethink:"boards"`
	Langs            []string      `json:"langs" gorethink:"langs"`
	Links            [][2]string   `json:"links" gorethink:"links"`
	Spoilers         spoilers      `json:"spoilers" gorethink:"spoliers"`
	SessionExpiry    time.Duration `json:"-" gorethink:"sessionExpiry"`
}

// Need a custom json.Marshaler, because []uint8 decodes the same as []byte by
// default
type spoilers []uint8

func (s spoilers) MarshalJSON() ([]byte, error) {
	buf := []byte{'['}
	for i, sp := range s {
		if i != 0 {
			buf = append(buf, ',')
		}
		buf = strconv.AppendUint(buf, uint64(sp), 10)
	}
	return append(buf, ']'), nil
}

// Defaults contains the default server configuration values
var Defaults = Configs{
	Prune:            false,
	WebmAudio:        true,
	Hats:             false,
	Radio:            false,
	MaxThreads:       100,
	MaxBump:          1000,
	JPEGQuality:      90,
	PNGQuality:       20,
	MaxSize:          3145728,
	MaxHeight:        6000,
	MaxWidth:         6000,
	ThreadCooldown:   60,
	MaxSubjectLength: 50,
	SessionExpiry:    30,
	ExcludeRegex:     "/[\u2000-\u200f\u202a-\u202f\u205f-\u206f]+/g",
	Origin:           "localhost:8000",
	Frontpage:        "",
	DefaultCSS:       "moe",
	Salt:             "LALALALALALALALALALALALALALALALALALALALA",
	FeedbackEmail:    "admin@email.com",
	FAQ:              defaultFAQ,
	DefaultLang:      "en_GB",
	Spoilers:         spoilers{0},
	Langs:            []string{"en_GB"},
	Boards:           []string{},
	Links:            [][2]string{{"4chan", "http://www.4chan.org/"}},
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
func Set(c Configs) error {
	client, err := json.Marshal(c)
	if err != nil {
		return err
	}
	h := util.HashBuffer(client)

	globalMu.Lock()
	defer globalMu.Unlock()
	clientJSON = client
	global = &c
	hash = h
	return nil
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
