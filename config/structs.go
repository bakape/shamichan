package config

// Configs stores the global server configuration
type Configs struct {
	Public
	PruneBoards         bool   `json:"pruneBoards"`
	HideNSFW            bool   `json:"hideNSFW"`
	EmailErr            bool   `json:"emailErr"`
	JPEGThumbnails      bool   `json:"JPEGThumbnails"`
	MaxWidth            uint16 `json:"maxWidth"`
	MaxHeight           uint16 `json:"maxHeight"`
	BoardExpiry         uint   `json:"boardExpiry"`
	SessionExpiry       uint   `json:"sessionExpiry"`
	EmailErrPort        uint   `json:"emailErrPort"`
	CharScore           uint   `json:"charScore"`
	PostCreationScore   uint   `json:"postCreationScore"`
	ImageScore          uint   `json:"imageScore"`
	RootURL             string `json:"rootURL"`
	Salt                string `json:"salt"`
	EmailErrMail        string `json:"emailErrMail"`
	EmailErrPass        string `json:"emailErrPass"`
	EmailErrSub         string `json:"emailErrSub"`
	FeedbackEmail       string `json:"feedbackEmail"`
	FAQ                 string
	CaptchaTags         []string          `json:"captchaTags"`
	OverrideCaptchaTags map[string]string `json:"overrideCaptchaTags"`
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
	Eightball     []string `json:"eightball"`
}

// BoardPublic contains publically accessible board-specific configurations
type BoardPublic struct {
	ReadOnly   bool `json:"readOnly"`
	TextOnly   bool `json:"textOnly"`
	ForcedAnon bool `json:"forcedAnon"`
	Flags      bool `json:"flags"`
	NSFW       bool
	RbText     bool   `json:"rbText"`
	Pyu        bool   `json:"pyu"`
	DefaultCSS string `json:"defaultCSS"`
	Title      string `json:"title"`
	Notice     string `json:"notice"`
	Rules      string `json:"rules"`

	// Can't use []uint8, because it marshals to string
	Banners []uint16 `json:"banners"`
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
