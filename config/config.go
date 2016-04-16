// Package config parses JSON configuration files and exports the configuration
// for server-side use and the public availability JSON struct, which includes
// a small subset of the server configuration.
package config

import (
	"encoding/json"
	"github.com/DisposaBoy/JsonConfigReader"
	"github.com/Soreil/mnemonics"
	"github.com/bakape/meguca/util"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

var (
	// Overridable path for tests
	configRoot = "config"

	// Ensures no reads happen, while the configuration is reloading
	mu sync.RWMutex

	// Contains currently loaded server configuration
	config ServerConfigs

	// JSON of client-accessable configuration
	clientJSON []byte

	// Hash of the config file. Used live updating configuration on the client
	hash string
)

// ServerConfigs stores the global configuration
type ServerConfigs struct {
	HTTP       HTTPConfigs
	Rethinkdb  RethinkDBConfig
	Boards     BoardConfig
	Staff      StaffConfig
	Images     ImageConfig
	Posts      PostConfig
	Recaptcha  RecaptchaConfig
	Radio, Pyu bool
	InfoBanner string
}

// HTTPConfigs stores HTTP server configuration
type HTTPConfigs struct {
	SSL, TrustProxies, Gzip bool
	Addr, Origin, Cert, Key string
	Frontpage               string
}

// RethinkDBConfig stores address and datbase name to connect to
type RethinkDBConfig struct {
	Addr, Db string
}

// BoardConfig stores overall board configuration
type BoardConfig struct {
	Enabled []string
	Boards  map[string]struct {
		MaxThreads, MaxBump int
		Title               string
	}
	Default, Staff string
	Psuedo, Links  [][2]string
	Prune          bool
}

// StaffConfig stores moderation staff related configuration
type StaffConfig struct {
	Classes     map[string]StaffClass
	SessionTime int
}

// StaffClass contains properties of a single staff personel type
type StaffClass struct {
	Alias   string
	Members map[string]string
	Rights  map[string]bool
}

// ImageConfig stores file upload processing and thumbnailing configuration
type ImageConfig struct {
	WebmAudio          bool
	Hats               bool
	JpegQuality        uint8
	DuplicateThreshold uint8
	Max                struct {
		Size          int64
		Width, Height int
	}
	Spoilers   []uint8
	PngQuality string
}

// PostConfig stores configuration related to creating posts
type PostConfig struct {
	ThreadCreationCooldown, MaxSubjectLength int
	ReadOnly, SageEnabled, ForcedAnon        bool
	Salt, ExcludeRegex                       string
}

// RecaptchaConfig stores the public and private key fot Google ReCaptcha
// authentication
type RecaptchaConfig struct {
	Public, Private string
}

// A subset of ServerConfigs, that is exported as JSON to all clients
type clientConfigs struct {
	Boards struct {
		Enabled []string `json:"enabled"`
		Boards  map[string]struct {
			Title string `json:"title"`
		} `json:"boards"`
		Default string      `json:"default"`
		Psuedo  [][2]string `json:"psuedo"`
		Links   [][2]string `json:"links"`
	} `json:"boards"`
	Lang struct {
		Enabled []string `json:"enabled"`
		Default string   `json:"default"`
	} `json:"lang"`
	Staff struct {
		Classes map[string]struct {
			Alias  string          `json:"alias"`
			Rights map[string]bool `json:"rights"`
		} `json:"classes"`
	} `json:"staff"`
	Images struct {
		thumb struct {
			ThumbDims [2]int `json:"thumbDims"`
			MidDims   [2]int `json:"midDims"`
		}
		Spoilers []int `json:"spoilers"`
		Hats     bool  `json:"hats"`
	} `json:"images"`
	Banners       []string `json:"banners"`
	FAQ           []string `json:"FAQ"`
	Eightball     []string `json:"eightball"`
	Radio         bool     `json:"radio"`
	IllyaDance    bool     `json:"illiyaDance"`
	FeedbackEmail string   `json:"feedbackEmail"`
	DefaultCSS    string   `json:"defaultCSS"`
	InfoBanner    string   `json:"infoBanner"`
}

// LoadConfig reads and parses the JSON config file and thread-safely loads it
// into the server
func LoadConfig() error {
	var (
		tempServer ServerConfigs
		tempClient clientConfigs
		path       = filepath.FromSlash(configRoot + "/config.json")
	)

	file, err := os.Open(path)
	if err != nil {
		return util.WrapError("Error reading configuration file", err)
	}
	defer file.Close()

	// Strip comments
	buf, err := ioutil.ReadAll(JsonConfigReader.New(file))
	if err != nil {
		return parseError(err)
	}

	if err := json.Unmarshal(buf, &tempServer); err != nil {
		return parseError(err)
	}
	if err := json.Unmarshal(buf, &tempClient); err != nil {
		return parseError(err)
	}

	tempJSON, err := json.Marshal(tempClient)
	if err != nil {
		return parseError(err)
	}
	tempHash, err := util.HashBuffer(buf)
	if err != nil {
		return parseError(err)
	}

	mu.Lock()
	defer mu.Unlock()

	config = tempServer
	clientJSON = tempJSON
	hash = tempHash
	if err := mnemonic.SetSalt(config.Posts.Salt); err != nil {
		return err
	}

	return nil
}

func parseError(err error) error {
	return util.WrapError("Error parsing configuration file", err)
}

// HTTP returns HTTP server configuration
func HTTP() HTTPConfigs {
	mu.RLock()
	defer mu.RUnlock()
	return config.HTTP
}

// RethinkDB returns address and datbase name to connect to
func RethinkDB() RethinkDBConfig {
	mu.RLock()
	defer mu.RUnlock()
	return config.Rethinkdb
}

// Boards returns overall board configuration
func Boards() BoardConfig {
	mu.RLock()
	defer mu.RUnlock()
	return config.Boards
}

// EnabledBoards returns a slice of curently enabled boards
func EnabledBoards() []string {
	mu.RLock()
	defer mu.RUnlock()
	return config.Boards.Enabled
}

// Staff returns moderation staff related configuration
func Staff() StaffConfig {
	mu.RLock()
	defer mu.RUnlock()
	return config.Staff
}

// Images returns file upload processing and thumbnailing configuration
func Images() ImageConfig {
	mu.RLock()
	defer mu.RUnlock()
	return config.Images
}

// Posts returns configuration related to creating posts
func Posts() PostConfig {
	mu.RLock()
	defer mu.RUnlock()
	return config.Posts
}

// Recaptcha returns the public and private key fot Google ReCaptcha
// authentication
func Recaptcha() RecaptchaConfig {
	mu.RLock()
	defer mu.RUnlock()
	return config.Recaptcha
}

// Client returns punlic availability configuration JSON and a truncated
// configuration MD5 hash
func Client() ([]byte, string) {
	mu.RLock()
	defer mu.RUnlock()
	return clientJSON, hash
}

// Radio returns, if r-a-d.io integration is enabled
func Radio() bool {
	mu.RLock()
	defer mu.RUnlock()
	return config.Radio
}

// Pyu returns, if don't ask is enabled
func Pyu() bool {
	mu.RLock()
	defer mu.RUnlock()
	return config.Pyu
}

// Set sets the internal configuration struct. To be used onl in tests.
func Set(c ServerConfigs) {
	mu.Lock()
	defer mu.Unlock()
	config = c
}

// SetClient sets the client configuration JSON and hash. To be used only in
// tests.
func SetClient(json []byte, cHash string) {
	mu.Lock()
	defer mu.Unlock()
	clientJSON = json
	hash = cHash
}
