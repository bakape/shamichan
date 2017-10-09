//go:generate easyjson --all --no_std_marshalers $GOFILE

package config

// Configs stores the global server configuration
type Configs struct {
	Public
	PruneBoards   bool `json:"pruneBoards"`
	Pyu           bool `json:"pyu"`
	HideNSFW      bool `json:"hideNSFW"`
	JPEGQuality   uint8
	MaxWidth      uint16 `json:"maxWidth"`
	MaxHeight     uint16 `json:"maxHeight"`
	BoardExpiry   uint   `json:"boardExpiry"`
	SessionExpiry uint   `json:"sessionExpiry"`
	RootURL       string `json:"rootURL"`
	Salt          string `json:"salt"`
	FeedbackEmail string `json:"feedbackEmail"`
	FAQ           string
}

// Public contains configurations exposeable through public availability APIs
type Public struct {
	Captcha           bool              `json:"captcha"`
	Mature            bool              `json:"mature"`
	DisableUserBoards bool              `json:"disableUserBoards"`
	PruneThreads      bool              `json:"pruneThreads"`
	ThreadExpiryMin   uint              `json:"threadExpiryMin"`
	ThreadExpiryMax   uint              `json:"threadExpiryMax"`
	MaxSize           uint              `json:"maxSize"`
	DefaultLang       string            `json:"defaultLang"`
	DefaultCSS        string            `json:"defaultCSS"`
	ImageRootOverride string            `json:"imageRootOverride"`
	Links             map[string]string `json:"links"`
}

// BoardConfigs stores board-specific configuration
type BoardConfigs struct {
	BoardPublic
	DisableRobots bool     `json:"disableRobots"`
	ID            string   `json:"id"`
	Js            string   `json:"js"`
	Eightball     []string `json:"eightball"`
}

// BoardPublic contains publically accessible board-specific configurations
type BoardPublic struct {
	ReadOnly   bool `json:"readOnly"`
	TextOnly   bool `json:"textOnly"`
	ForcedAnon bool `json:"forcedAnon"`
	Flags      bool `json:"flags"`
	NonLive    bool `json:"nonLive"`
	NSFW       bool
	PosterIDs  bool   `json:"posterIDs"`
	DefaultCSS string `json:"defaultCSS"`
	Title      string `json:"title"`
	Notice     string `json:"notice"`
	Rules      string `json:"rules"`
}

// BoardConfContainer contains configurations for an individual board as well
// as pregenerated public JSON and it's hash
type BoardConfContainer struct {
	BoardConfigs
	JSON []byte
	Hash string
}

// BoardTitle contains a board's ID and title
type BoardTitle struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// BoardTitles implements sort.Interface
type BoardTitles []BoardTitle

func (b BoardTitles) Len() int {
	return len(b)
}

func (b BoardTitles) Less(i, j int) bool {
	return b[i].ID < b[j].ID
}

func (b BoardTitles) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}
