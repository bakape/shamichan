package common

import (
	"os"
	"regexp"
	"sort"
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
	CommandRegexp = regexp.MustCompile(`^#(flip|\d*d\d+|8ball|pyu|pcount|sw(?:\d+:)?\d+:\d+(?:[+-]\d+)?)$`)
	DiceRegexp    = regexp.MustCompile(`(\d*)d(\d+)`)
)

func init() {
	sort.Strings(Langs)
	sort.Strings(Themes)
}
