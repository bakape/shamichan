// Package config parses JSON configuration files and exports the configuration
// for server-side use and the public availability JSON struct, which includes
// a small subset of the server configuration.
package config

import (
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

	// AllBoardConfigs stores board-specific configurations for the /all/
	// metaboard. Constant.
	AllBoardConfigs []byte

	// JSON of client-accessable configuration
	clientJSON []byte

	// Hash of the gloabal configs. Used for live reloading configuration on the
	// client.
	hash string

	// AllowedOrigin stores the accepted client origin for websocket and file
	// upload requests. Set only on server start.
	AllowedOrigin string

	// Defaults contains the default server configuration values
	Defaults = Configs{
		MaxThreads:    100,
		MaxBump:       1000,
		JPEGQuality:   90,
		PNGQuality:    20,
		MaxSize:       5,
		MaxHeight:     6000,
		MaxWidth:      6000,
		SessionExpiry: 30,
		DefaultCSS:    "moe",
		Salt:          "LALALALALALALALALALALALALALALALALALALALA",
		FeedbackEmail: "admin@email.com",
		FAQ:           defaultFAQ,
		DefaultLang:   "en_GB",
		Boards:        []string{},
		Links:         map[string]string{"4chan": "http://www.4chan.org/"},
	}

	// EightballDefaults contains the default eightball answer set
	EightballDefaults = []string{
		"Yes",
		"No",
		"Maybe",
		"It can't be helped",
		"Hell yeah, motherfucker!",
		"Anta baka?",
	}
)

// Default string for the FAQ panel
const defaultFAQ = `Upload size limit is 5 MB
Accepted upload file types: JPG, JPEG, PNG, GIF, WEBM, SVG, PDF, MP3, MP4, OGG
<hr>Hash commands:
#d100 #2d100 - Roll dice
#flip - Coin flip
#8ball - An 8ball
#queue - Print r/a/dio song queue
#sw24:15 #sw2:24:15 #sw24:15+30 #sw24:15-30 - Syncronised duration timer

All hash commands need to be input on their own line`

// Configs stores the global configuration
type Configs struct {
	Prune             bool   `json:"prune" gorethink:"prune"`
	Radio             bool   `json:"radio" gorethink:"radio" public:"true"`
	Hats              bool   `json:"hats" gorethink:"hats" public:"true"`
	IllyaDance        bool   `json:"illyaDance" gorethink:"illyaDance" public:"true"`
	Pyu               bool   `json:"pyu" gorethink:"pyu"`
	Captcha           bool   `json:"captcha" gorethink:"captcha" public:"true"`
	MaxWidth          uint16 `json:"maxWidth" gorethink:"maxWidth"`
	MaxHeight         uint16 `json:"maxHeight" gorethink:"maxHeight"`
	MaxThreads        int    `json:"maxThreads" gorethink:"maxThreads"`
	MaxBump           int    `json:"maxBump" gorethink:"maxBump"`
	JPEGQuality       int
	PNGQuality        int
	MaxSize           int64             `json:"maxSize" gorethink:"maxSize"`
	DefaultLang       string            `json:"defaultLang" gorethink:"defaultLang" public:"true"`
	DefaultCSS        string            `json:"defaultCSS" gorethink:"defaultCSS" public:"true"`
	Salt              string            `json:"salt" gorethink:"salt"`
	FeedbackEmail     string            `json:"feedbackEmail" gorethink:"feedbackEmail"`
	FAQ               string            `public:"true"`
	CaptchaPublicKey  string            `json:"captchaPublicKey" gorethink:"captchaPublicKey" public:"true"`
	CaptchaPrivateKey string            `json:"captchaPrivateKey" gorethink:"captchaPrivateKey"`
	Boards            []string          `json:"-" gorethink:"boards" public:"true"`
	Links             map[string]string `json:"links" gorethink:"links" public:"true"`
	SessionExpiry     time.Duration     `json:"sessionExpiry" gorethink:"sessionExpiry"`
}

// Only marshal JSON with the `public:"true"` tag for publicly exposed
// configuration
func (c *Configs) marshalPublicJSON() ([]byte, error) {
	t := reflect.TypeOf(*c)
	v := reflect.ValueOf(*c)

	// Copy the fields we need to a map
	temp := make(map[string]interface{}, 10)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Tag.Get("public") != "true" {
			continue
		}

		name := t.Field(i).Tag.Get("gorethink")
		if name == "" {
			name = field.Name
		}
		temp[name] = v.Field(i).Interface()
	}

	return json.Marshal(temp)
}

// BoardConfigs stores board-specific configuration
type BoardConfigs struct {
	PostParseConfigs
	Spoilers  bool                `json:"spoilers" gorethink:"spoilers"`
	CodeTags  bool                `json:"codeTags" gorethink:"codeTags" public:"true"`
	ID        string              `json:"id" gorethink:"id"`
	Spoiler   string              `json:"spoiler" gorethink:"spoiler" public:"true"`
	Title     string              `json:"title" gorethink:"title" public:"true"`
	Notice    string              `json:"notice" gorethink:"notice" public:"true"`
	Eightball []string            `json:"eightball" gorethink:"eightball"`
	Banners   []string            `json:"banners" gorethink:"banners" public:"true"`
	Staff     map[string][]string `json:"staff" gorethink:"staff"`
}

// MarshalPublicJSON marshals the publically exposed fields of a board-specific
// configuration
func (b *BoardConfigs) MarshalPublicJSON() ([]byte, error) {
	t := reflect.TypeOf(*b)
	v := reflect.ValueOf(*b)

	// Convert all the fields of PostParseConfigs
	temp := b.PostParseConfigs.toMap(9)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Tag.Get("public") != "true" {
			continue
		}
		temp[t.Field(i).Tag.Get("json")] = v.Field(i).Interface()
	}

	return json.Marshal(temp)
}

// PostParseConfigs contains board-specific flags for post text parsing
type PostParseConfigs struct {
	ReadOnly     bool `json:"readOnly" gorethink:"readOnly"`
	TextOnly     bool `json:"textOnly" gorethink:"textOnly"`
	ForcedAnon   bool `json:"forcedAnon" gorethink:"forcedAnon"`
	HashCommands bool `json:"hashCommands" gorethink:"hashCommands"`
}

// Converts p to a a map[string]interface{} of desired length
func (p PostParseConfigs) toMap(length int) map[string]interface{} {
	t := reflect.TypeOf(p)
	v := reflect.ValueOf(p)
	m := make(map[string]interface{}, length)

	for i := 0; i < t.NumField(); i++ {
		m[t.Field(i).Tag.Get("json")] = v.Field(i).Interface()
	}

	return m
}

// Generate /all/ board configs
func init() {
	conf := BoardConfigs{
		PostParseConfigs: PostParseConfigs{
			HashCommands: true,
		},
		Spoilers: true,
		CodeTags: true,
		Spoiler:  "default.jpg",
		Title:    "Aggregator metaboard",
		Banners:  []string{},
	}

	var err error
	AllBoardConfigs, err = conf.MarshalPublicJSON()
	if err != nil {
		panic(err)
	}
}

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
