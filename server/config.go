/*
 Parses JSON configuration files and exports the config struct for server-side
 use and the clientConfig struct, for JSON stringification and passing to the
 client
*/

package server

import (
	"io/ioutil"
	"os"
)

// config contains currently loaded configuration
var config struct {
	// Configuration that can not be hot-reloaded without restarting the server
	Hard struct {
		HTTP struct {
			Addr, Origin             string
			TrustProxies, Websockets bool
		}
		Redis struct {
			Addr string
			Db   int
		}
		Rethinkdb struct {
			Addr, Db string
		}
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
		Classes map[string]struct {
			Alias   string
			Members map[string]string
			Rights  map[string]bool
		}
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
		Spoilers           []uint16
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
	Debug                                                          bool
	Banners, FAQ, Eightball                                        []string
	Schedule                                                       [][3]string
	Radio, Pyu, IllyaDance                                         bool
	FeedbackEmail, DefaultCSS, Frontpage, InfoBanner, InjectJSPath string
}

// clientConfig exports public settings all clients can access
var clientConfig struct {
	Hard struct {
		HTTP struct {
			Websockets bool `json:"websockets"`
		} `json:"HTTP"`
	} `json:"hard"`
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
	Debug         bool        `json:"debug"`
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

// configHash is the truncated MD5 hash of the JSON configuration file
var configHash string

// loadConfig reads and parses JSON config files
func loadConfig() {
	const path = "./config/config.json"
	file, err := ioutil.ReadFile(path)

	// If config file does not exist, read and copy defaults file
	if err != nil {
		if os.IsNotExist(err) {
			file, err = ioutil.ReadFile("./config/defaults.json")
			throw(err)
			throw(ioutil.WriteFile(path, file, 0600))
		} else {
			panic(err)
		}
	}

	unmarshalJSON(file, &config)
	unmarshalJSON(file, &clientConfig)
	configHash = hashBuffer(file)
}
