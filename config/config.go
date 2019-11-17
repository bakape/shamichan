// Package config stores and exports the configuration for server-side use and
// the public availability JSON struct, which includes a small subset of the
// server configuration.
package config

import (
	"encoding/json"
	"sync"

	"github.com/bakape/meguca/util"
)

var (
	// Ensures no reads happen, while the configuration is reloading
	globalMu, boardMu sync.RWMutex

	// Contains currently loaded global server configuration
	global *Configs

	// JSON of client-accessible configuration
	clientJSON []byte

	// Hash of the global configs. Used for live reloading configuration on the
	// client.
	hash string

	// Defaults contains the default server configuration values
	Defaults = Configs{
		MaxHeight:         6000,
		MaxWidth:          6000,
		CharScore:         170,
		PostCreationScore: 15000,
		ImageScore:        15000,
		EmailErrPort:      587,
		Salt:              "LALALALALALALALALALALALALALALALALALALALA",
		RootURL:           "http://127.0.0.1",
		FAQ:               defaultFAQ,
		CaptchaTags: []string{
			"patchouli_knowledge",
			"cirno",
			"hakurei_reimu",
		},
		Public: Public{
			DefaultCSS:   "moe",
			DefaultLang:  "en_GB",
			ThreadExpiry: 7,
			MaxSize:      5,
			Links:        map[string]string{"4chan": "http://www.4chan.org/"},
		},
	}
)

// Default string for the FAQ panel
const defaultFAQ = `Supported upload file types are JPEG, PNG, APNG, WEBM, MP3, FLAC, MP4, OGG, PDF, ZIP, 7Z, TAR.GZ, TAR.XZ, RAR, CBZ, CBR.
<hr>Encase text in:
  ** for spoilers
  @@ for bold
  ~~ for italics
  ^r for red text
  ^b for blue text
  ` + "``" + ` for programing code highlighting
<hr>Hash commands:
#d100 #2d100 - Roll dice
#flip - Coin flip
#8ball - An 8ball
#sw24:30 #sw2:24:30 #sw24:30+30 #sw24:30-30 - "Syncwatch" synchronized time counter`

// Get returns a pointer to the current server configuration struct. Callers
// should not modify this struct.
func Get() *Configs {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return global
}

// Set sets the internal configuration struct
func Set(c Configs) (err error) {
	client, err := json.Marshal(c.Public)
	if err != nil {
		return
	}
	h := util.HashBuffer(client)

	globalMu.Lock()
	clientJSON = client
	global = &c
	hash = h
	globalMu.Unlock()

	return util.Trigger("config.changed")
}

// GetClient returns public availability configuration JSON and a truncated
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
	clientJSON = json
	hash = cHash
	globalMu.Unlock()
}

// Clear resets package state. Only use in tests.
func Clear() {
	globalMu.Lock()
	defer globalMu.Unlock()

	global = &Configs{}
	clientJSON = nil
	hash = ""
}
