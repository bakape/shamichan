/*
 Parses JSON configuration files and exports the config struct for server-side
 use and the clientConfig struct, for JSON stringification and passing to the
 client
*/

package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
)

// config contains currently loaded configuration
var config struct {
	// Configuration that can not be hot-reloaded without restarting the server
	Hard struct {
		HTTP struct {
			Port                                                     int
			Host, Media, Upload, Socket, Origin                      string
			ServeStatic, ServeImages, TrustProxies, Gzip, Websockets bool
		}
		Redis struct {
			Addr string
			Db   int
		}
		Rethinkdb struct {
			Addr, Db string
		}
		Dirs struct {
			Src, Thumb, Mid, Tmp string
		}
		Debug bool
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
		Def     string
	}
	Staff struct {
		Enabled     map[string]map[string]string
		Aliases     map[string]string
		Keyword     string
		SessionTime int
	}
	Images struct {
		Max struct {
			Size, Width, Height, Pixels int
		}
		Thumb struct {
			Quality              int
			SmallDims, ThumbDims [2]int
			HighQuality, PNG     bool
			PNGQuality           string
		}
		Formats struct {
			Webm, WebmAudio, MP3, SVG, PDF bool
		}
		DuplicateThreshold int
		Spoilers           []int
		Hats               bool
	}
	Posts struct {
		Salt, ExcludeRegex                              string
		ThreadCreationCooldown, LastN, MaxSubjectLength int
		ReadOnly, SageEnabled, ForcedAnon               bool
	}
	Recaptcha struct {
		Public, Private string
	}
	Banners, FAQ, Eightball                                        []string
	Schedule                                                       [][3]string
	Radio, Pyu, IllyaDance                                         bool
	FeedbackEmail, DefaultCSS, Frontpage, InfoBanner, InjectJSPath string
}

// clientConfig exports public settings client can access
var clientConfig struct {
	Hard struct {
		HTTP struct {
			Media      string `json:"media"`
			Upload     string `json:"upload"`
			Socket     string `json:"socket"`
			Websockets bool   `json:"websockets"`
		} `json:"HTTP"`
		Debug bool `json:"debug"`
	} `json:"hard"`
	Boards struct {
		Enabled []string `json:"enabled"`
		Boards  map[string]struct {
			Title string `json:"title"`
		} `json:"boards"`
		Default string      `json:"def"`
		Psuedo  [][2]string `json:"psuedo"`
		Links   [][2]string `json:"links"`
	} `json:"boards"`
	Lang struct {
		Enabled []string `json:"enabled"`
		Def     string   `json:"def"`
	} `json:"lang"`
	Staff struct {
		Aliases map[string]string `json:"aliases"`
		Keyword string            `json:"keyword"`
	} `json:"staff"`
	Images struct {
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

// configHash is the truncated MD5 hash of the JSON configuration file
var configHash string

// loadConfig reads and parses JSON config files
func loadConfig() {
	file, err := ioutil.ReadFile("./config/config.json")
	throw(err)
	throw(json.Unmarshal(file, &config))
	throw(json.Unmarshal(file, &clientConfig))
	configHash = hex.EncodeToString(md5.New().Sum(file))[:16]
}
