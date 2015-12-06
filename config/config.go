package config

import (
	"encoding/json"
	"io/ioutil"
)

type config struct {
	// Configuration that can not be hot-reloaded without restarting the server
	hard struct {
		http struct {
			port                                                     int
			host, media, upload, socket, origin                      string
			serveStatic, serveImages, trustProxies, gzip, websockets bool
		}
		redis struct {
			port, db int
			host     string
		}
		rethinkdb struct {
			port     int
			host, db string
		}
		dirs struct {
			src, thumb, mid, tmp string
		}
		debug bool
	}
	boards struct {
		enabled map[string]struct {
			maxThreads, maxBump int
			title               string
		}
		def, staff   string
		psuedo, link [2]string
		prune        bool
	}
	lang struct {
		enabled []string
		def     string
	}
	staff struct {
		enabled     map[string]map[string]string
		aliases     map[string]string
		keyword     string
		sessionTime int
	}
	images struct {
		max struct {
			size, width, height, pixels int
		}
		thumb struct {
			quality              int
			smallDims, thumbDims [2]int
			highQuality, png     bool
			pngQuality           string
		}
		formats struct {
			webm, webmAudio, mp3, svg, pdf bool
		}
		duplicateThreshold int
		spoilers           []int
		hats               bool
	}
	posts struct {
		salt, excludeRegex                              string
		threadCreationCooldown, lastN, maxSubjectLength int
		readOnly, sageEnabled, forcedAnon               bool
	}
	recaptcha struct {
		public, private string
	}
	banners, FAQ, eightball                                        []string
	schedule                                                       [][3]string
	radio, pyu, illyaDance                                         bool
	feedbackEmail, defaultCSS, frontpage, infoBanner, injectJSPath string
}

// Config contains currently loaded configuration
var Config config

// Load reads and parses JSON config files
func Load() error {
	file, err := ioutil.ReadFile("./config/defaults.json")
	if err != nil {
		return err
	}
	var conf config
	json.Unmarshal(file, &conf)
	Config = conf
	return nil
}
