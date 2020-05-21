package config

// Amounts to increase spam score by for a user action
type SpamScores struct {
	Char         uint64 `json:"character"`
	Image        uint64 `json:"image"`
	PostCreation uint64 `json:"post_creation"`
}

// Configs stores the global server configuration
type Configs struct {
	Public
	EmailErr       bool   `json:"email_errors"`
	JPEGThumbnails bool   `json:"JPEG_thumbnails"`
	DisableRobots  bool   `json:"disable_robots"`
	MaxWidth       uint16 `json:"max_width"`
	MaxHeight      uint16 `json:"max_height"`
	EmailErrPort   uint   `json:"email_errors_server_port"`
	RootURL        string `json:"root_URL"`
	Salt           string `json:"salt"`
	EmailErrMail   string `json:"email_errors_address"`
	EmailErrPass   string `json:"email_errors_password"`
	EmailErrSub    string `json:"email_errors_server_address"`
	FAQ            string
	CaptchaTags    []string   `json:"captcha_tags"`
	SpamScores     SpamScores `json:"spam_scores"`
}

// Public contains configurations exposeable through public availability APIs
type Public struct {
	Captcha           bool              `json:"captcha"`
	Mature            bool              `json:"mature"`
	PruneThreads      bool              `json:"prune_threads"`
	ThreadExpiry      uint              `json:"thread_expiry"`
	MaxSize           uint              `json:"max_size"`
	DefaultLang       string            `json:"default_lang"`
	DefaultCSS        string            `json:"default_theme"`
	ImageRootOverride string            `json:"image_root_override"`
	Links             map[string]string `json:"links"`
}
