package common

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/lang"
)

var (
	// GetVideoNames is a forwarded function
	// from "github.com/bakape/megucaassets" to avoid circular imports
	GetVideoNames func() []string
	// Recompile is a forwarded function
	// from "github.com/bakape/megucatemplates" to avoid circular imports
	Recompile func() error

	// Project is being unit tested
	IsTest bool

	// Currently running inside CI
	IsCI = os.Getenv("CI") == "true"
)

// Maximum lengths of various input fields
const (
	MaxLenName         = 50
	MaxLenAuth         = 50
	MaxLenPostPassword = 100
	MaxLenSubject      = 100
	MaxLenBody         = 2000
	MaxLinesBody       = 100
	MaxLenPassword     = 50
	MaxLenUserID       = 20
	MaxLenBoardID      = 10
	MaxLenBoardTitle   = 100
	MaxLenNotice       = 500
	MaxLenRules        = 5000
	MaxLenEightball    = 2000
	MaxLenReason       = 100
	MaxNumBanners      = 20
	MaxAssetSize       = 100 << 10
	MaxDiceSides       = 10000
	BumpLimit          = 1000
)

// Various cryptographic token exact lengths
const (
	LenSession    = 171
	LenImageToken = 86
)

// Available language packs and themes. Change this, when adding any new ones.
var (
	Langs = []string{
		"en_GB",
		"es_ES",
		"fr_FR",
		"nl_NL",
		"pl_PL",
		"pt_BR",
		"ru_RU",
		"sk_SK",
		"tr_TR",
		"uk_UA",
		"zh_TW",
	}
	Themes = []string{
		"ashita",
		"console",
		"egophobe",
		"gar",
		"glass",
		"gowno",
		"higan",
		"inumi",
		"mawaru",
		"moe",
		"moon",
		"ocean",
		"rave",
		"tavern",
		"tea",
		"win95",
	}
)

// Common Regex expressions
var (
	CommandRegexp = regexp.MustCompile(`^#(flip|\d*d\d+|8ball|pyu|pcount|sw(?:\d+:)?\d+:\d+(?:[+-]\d+)?|roulette|rcount)$`)
	DiceRegexp    = regexp.MustCompile(`(\d*)d(\d+)`)

	DomainRegexp = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{1,61}[a-zA-Z0-9]\.[a-zA-Z]{2,}$`)

	invidiousUrlRegexpStr = `https?:\/\/(?:www\.)?invidio\.us\/watch(?:.*&|\?)v=(.+)(?:\?.+)*`
	InvidiousUrlRegexp = regexp.MustCompile(invidiousUrlRegexpStr)
	youtubeUrlRegexpStr = `https?:\/\/(?:www\.)?youtube\.com\/watch(?:.*&|\?)v=(.+)(?:\?.+)*`
	YoutubeUrlRegexp = regexp.MustCompile(youtubeUrlRegexpStr)
	updateMutex sync.RWMutex
	rawVideoUrlRegexp *regexp.Regexp
	cinemaPushRegexp *regexp.Regexp
	cinemaSkipRegexp *regexp.Regexp
)

func GetRawVideoUrlRegexp() *regexp.Regexp {
	updateMutex.RLock()
	defer updateMutex.RUnlock()
	return rawVideoUrlRegexp
}

func GetCinemaPushRegexp() *regexp.Regexp {
	updateMutex.RLock()
	defer updateMutex.RUnlock()
	return cinemaPushRegexp
}

func GetCinemaSkipRegexp() *regexp.Regexp {
	updateMutex.RLock()
	defer updateMutex.RUnlock()
	return cinemaSkipRegexp
}

func Update() (err error) {
	updateMutex.Lock()
	defer updateMutex.Unlock()
	cinemaSources := []string{invidiousUrlRegexpStr, youtubeUrlRegexpStr}

	youtubeUrlRegexpStr := `https?:\/\/(?:www\.)?youtube\.com\/watch(?:.*&|\?)v=(.+)(?:\?.+)*`
	YoutubeUrlRegexp = regexp.MustCompile(youtubeUrlRegexpStr)
	cinemaSources = append(cinemaSources, youtubeUrlRegexpStr)

	conf := config.Get()
	if len(conf.CinemaRawDomains) > 0 {
		rawVideoUrlRegexpStr := `https?:\/\/(?:www\.)?[^\/]*(` +
			strings.ReplaceAll(strings.Join(conf.CinemaRawDomains, `|`), `.`, `\.`) +
			`)\/.*\.(webm|mp4|ogg|ogv)`
		rawVideoUrlRegexp = regexp.MustCompile(rawVideoUrlRegexpStr)
		cinemaSources = append(cinemaSources, rawVideoUrlRegexpStr)
	} else {
		rawVideoUrlRegexp = regexp.MustCompile(`^\b$`) // hacky way to never match
	}

	ln := lang.Get()
	cinemaPushRegexp = regexp.MustCompile(`^!` + ln.UI["cinemaPush"] +
		` (`+ strings.Join(cinemaSources, `|`) +`)$`)
	cinemaSkipRegexp =  regexp.MustCompile(`^!` + ln.UI["cinemaSkip"] + `$`)

	return
}

func init() {
	sort.Strings(Langs)
	sort.Strings(Themes)
}
