// Package config parses JSON configuration files and exports the Config struct
// for server-side use and the ClientConfig struct, for JSON stringification and
// passing to the  client,
package config

import (
	"github.com/Soreil/mnemonics"
	"github.com/bakape/meguca/util"
	"io/ioutil"
	"os"
)

// Overridable path for tests
var configRoot = "./config"

// Server stores the global configuration. It is loaded only once
// during start up and considered implicitly immutable during the rest of
// runtime.
type Server struct {
	HTTP struct {
		Addr, Domain, Cert, Key string
		SSL, TrustProxies       bool
	}
	Rethinkdb struct {
		Addr, Db string
	}
	Boards struct {
		Enabled []string
		Boards  map[string]struct {
			MaxThreads, MaxBump int
			Title               string
		}
		Default, Staff string
		Psuedo, Links  [][2]string
		Prune          bool
	}
	Lang struct {
		Enabled []string
		Default string
	}
	Staff struct {
		Classes     map[string]StaffClass
		Keyword     string
		SessionTime int
	}
	Images struct {
		Max struct {
			Size, Width, Height, Pixels int64
		}
		Thumb struct {
			HighQuality, PNG   bool
			Quality            int
			ThumbDims, MidDims [2]int
			PNGQuality         string
		}
		WebmAudio          bool
		Hats               bool
		DuplicateThreshold uint8
		Spoilers           []uint8
		Formats            map[string]bool
	}
	Posts struct {
		Salt, ExcludeRegex                       string
		ThreadCreationCooldown, MaxSubjectLength int
		ReadOnly, SageEnabled, ForcedAnon        bool
	}
	Recaptcha struct {
		Public, Private string
	}
	Banners, FAQ, Eightball                                        []string
	Schedule                                                       [][3]string
	Radio, Pyu, IllyaDance                                         bool
	FeedbackEmail, DefaultCSS, Frontpage, InfoBanner, InjectJSPath string
}

// StaffClass contains properties of a single staff personel type
type StaffClass struct {
	Alias   string
	Members map[string]string
	Rights  map[string]bool
}

// Config contains currently loaded server configuration
var Config Server

// client is a subset of serverConfigs, that is exported as JSON to all clients
type client struct {
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
		Keyword string `json:"keyword"`
	} `json:"staff"`
	Images struct {
		thumb struct {
			ThumbDims [2]int `json:"thumbDims"`
			MidDims   [2]int `json:"midDims"`
		}
		Spoilers []int `json:"spoilers"`
		Hats     bool  `json:"hats"`
	} `json:"images"`
	Banners       []string    `json:"banners"`
	FAQ           []string    `json:"FAQ"`
	Eightball     []string    `json:"eightball"`
	Schedule      [][3]string `json:"schedule"`
	Radio         bool        `json:"radio"`
	IllyaDance    bool        `json:"illiyaDance"`
	FeedbackEmail string      `json:"feedbackEmail"`
	DefaultCSS    string      `json:"defaultCSS"`
	InfoBanner    string      `json:"infoBanner"`
}

// ClientConfig exports public settings all clients can access
var ClientConfig []byte

// Hash stores the truncated MD5 hash of Config
var Hash string

// LoadConfig reads and parses the JSON config file
func LoadConfig() {
	path := configRoot + "/config.json"
	file, err := ioutil.ReadFile(path)

	// If config file does not exist, read and copy defaults file
	if err != nil {
		if os.IsNotExist(err) {
			file, err = ioutil.ReadFile(configRoot + "/defaults.json")
			util.Throw(err)
			util.Throw(ioutil.WriteFile(path, file, 0600))
		} else {
			panic(err)
		}
	}

	util.UnmarshalJSON(file, &Config)
	var data client
	util.UnmarshalJSON(file, &data)
	ClientConfig = util.MarshalJSON(data)
	Hash = util.HashBuffer(file)
	util.Throw(mnemonic.SetSalt(Config.Posts.Salt))
}
