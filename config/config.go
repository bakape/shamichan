// Package config parses JSON configuration files and exports the configuration
// for server-side use and the public availability JSON struct, which includes
// a small subset of the server configuration.
package config

import (
	"bytes"
	"encoding/json"
	"reflect"
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
	Prune            bool   `json:"prune" gorethink:"prune"`
	Radio            bool   `json:"radio" gorethink:"radio" public:"true"`
	Hats             bool   `json:"hats" gorethink:"hats" public:"true"`
	IllyaDance       bool   `json:"illyaDance" gorethink:"illyaDance" public:"true"`
	Pyu              bool   `json:"pyu" gorethink:"pyu"`
	MaxWidth         uint16 `json:"maxWidth" gorethink:"maxWidth"`
	MaxHeight        uint16 `json:"maxHeight" gorethink:"maxHeight"`
	MaxThreads       int    `json:"maxThreads" gorethink:"maxThreads"`
	MaxBump          int    `json:"maxBump" gorethink:"maxBump"`
	JPEGQuality      int
	PNGQuality       int
	ThreadCooldown   int           `json:"threadCooldown" gorethink:"threadCooldown" public:"true"`
	MaxSubjectLength int           `json:"maxSubjectLength" gorethink:"maxSubjectLength" public:"true"`
	MaxSize          int64         `json:"maxSize" gorethink:"maxSize"`
	DefaultLang      string        `json:"defaultLang" gorethink:"defaultLang" public:"true"`
	Origin           string        `json:"origin" gorethink:"origin"`
	DefaultCSS       string        `json:"defaultCSS" gorethink:"defaultCSS" public:"true"`
	Salt             string        `json:"salt" gorethink:"salt"`
	ExcludeRegex     string        `json:"excludeRegex" gorethink:"excludeRegex"`
	FeedbackEmail    string        `json:"feedbackEmail" gorethink:"feedbackEmail"`
	FAQ              string        `public:"true"`
	Boards           []string      `json:"-" gorethink:"boards" public:"true"`
	Links            [][2]string   `json:"links" gorethink:"links" public:"true"`
	SessionExpiry    time.Duration `json:"sessionExpiry" gorethink:"sessionExpiry"`
}

// Only marshal JSON with the `public:"true"` tag for publicly exposed
// configuration
func (c *Configs) marshalPublicJSON() ([]byte, error) {
	buf := new(bytes.Buffer)
	t := reflect.TypeOf(*c)
	v := reflect.ValueOf(*c)
	var notFirst bool

	buf.WriteByte('{')
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Tag.Get("public") != "true" {
			continue
		}

		name := t.Field(i).Tag.Get("gorethink")
		if name == "" {
			name = field.Name
		}

		if notFirst {
			buf.WriteByte(',')
		}
		buf.WriteByte('"')
		buf.WriteString(name)
		buf.WriteString(`":`)

		data, err := json.Marshal(v.Field(i).Interface())
		if err != nil {
			return nil, err
		}
		buf.Write(data)

		notFirst = true
	}
	buf.WriteByte('}')

	return buf.Bytes(), nil
}

// Defaults contains the default server configuration values
var Defaults = Configs{
	Prune:            false,
	Hats:             false,
	Radio:            false,
	MaxThreads:       100,
	MaxBump:          1000,
	JPEGQuality:      80,
	PNGQuality:       20,
	MaxSize:          3145728,
	MaxHeight:        6000,
	MaxWidth:         6000,
	ThreadCooldown:   60,
	MaxSubjectLength: 50,
	SessionExpiry:    30,
	ExcludeRegex:     "/[\u2000-\u200f\u202a-\u202f\u205f-\u206f]+/g",
	Origin:           "localhost:8000",
	DefaultCSS:       "moe",
	Salt:             "LALALALALALALALALALALALALALALALALALALALA",
	FeedbackEmail:    "admin@email.com",
	FAQ:              defaultFAQ,
	DefaultLang:      "en_GB",
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
	client, err := c.marshalPublicJSON()
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
