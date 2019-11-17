package config

// Configs stores the global server configuration
type Configs struct {
	Public
	EmailErr            bool   `json:"email_errors"`
	JPEGThumbnails      bool   `json:"JPEG_thumbnails"`
	GlobalDisableRobots bool   `json:"disable_robots"`
	MaxWidth            uint16 `json:"max_width"`
	MaxHeight           uint16 `json:"max_height"`
	EmailErrPort        uint   `json:"email_errors_server_port"`
	CharScore           uint   `json:"char_score"`
	PostCreationScore   uint   `json:"post_creation_score"`
	ImageScore          uint   `json:"image_score"`
	RootURL             string `json:"root_URL"`
	Salt                string `json:"salt"`
	EmailErrMail        string `json:"email_errors_address"`
	EmailErrPass        string `json:"email_errors_password"`
	EmailErrSub         string `json:"email_errors_server_address"`
	FeedbackEmail       string `json:"feedback_email"`
	FAQ                 string
	CaptchaTags         []string `json:"captcha_tags"`
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
