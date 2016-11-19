// Specifications for various input elements

package templates

var (
	// Available language packs. Change this, when adding any new ones.
	langs = []string{
		"en_GB", "es_ES", "pt_BR", "sk_SK", "tr_TR", "uk_UA",
	}

	// Available themes. Change this, when adding any new ones.
	themes = []string{
		"moe", "gar", "mawaru", "moon", "ashita", "console", "tea",
		"higan", "ocean", "rave", "glass",
	}
)

var specs = map[string][]inputSpec{
	"identity": {
		{
			ID:        "name",
			Type:      _string,
			MaxLength: 50,
		},
		{
			ID:        "email",
			Type:      _string,
			MaxLength: 100,
		},
		{
			ID:        "postPassword",
			Type:      _password,
			MaxLength: 50,
			Required:  true,
		},
	},
	"login": {
		{
			ID:        "id",
			Type:      _string,
			MaxLength: 20,
			NoID:      true,
		},
		{
			ID:        "password",
			Type:      _password,
			MaxLength: 30,
			NoID:      true,
		},
	},
	"register": {
		{
			ID:        "id",
			Type:      _string,
			MaxLength: 20,
			NoID:      true,
		},
		{
			ID:        "password",
			Type:      _password,
			MaxLength: 30,
			NoID:      true,
		},
		{
			ID:        "repeat",
			Type:      _password,
			MaxLength: 30,
			NoID:      true,
		},
	},
	"configureBoard": {
		{ID: "readOnly"},
		{ID: "textOnly"},
		{ID: "forcedAnon"},
		{ID: "hashCommands"},

		// TODO: Code tags

		{
			ID:        "title",
			Type:      _string,
			MaxLength: 100,
		},
		{
			ID:        "notice",
			Type:      _textarea,
			Rows:      5,
			MaxLength: 500,
		},
		{
			ID:        "rules",
			Type:      _textarea,
			Rows:      5,
			MaxLength: 5000,
		},
		{
			ID:        "eightball",
			Type:      _textarea,
			Rows:      5,
			MaxLength: 2000,
		},

		// TODO: Banner upload
		// TODO: Staff configuration

	},
	"createBoard": {
		{
			ID:        "boardName",
			Type:      _string,
			Required:  true,
			Pattern:   "^[a-z0-9]{1,3}$",
			MaxLength: 3,
		},
		{
			ID:        "boardTitle",
			Type:      _string,
			Required:  true,
			MaxLength: 100,
		},
	},
	"configureServer": {
		{ID: "mature"},
		{ID: "pruneThreads"},
		{
			ID:       "threadExpiry",
			Type:     _number,
			Min:      1,
			Required: true,
		},
		{ID: "pruneBoards"},
		{
			ID:       "boardExpiry",
			Type:     _number,
			Min:      1,
			Required: true,
		},
		{
			ID:   "salt",
			Type: _string,
		},
		{ID: "captcha"},
		{
			ID:   "captchaPublicKey",
			Type: _string,
		},
		{
			ID:   "captchaPrivateKey",
			Type: _string,
		},
		{
			ID:       "sessionExpiry",
			Type:     _number,
			Min:      1,
			Required: true,
		},
		{
			ID:   "feedbackEmail",
			Type: _string,
		},
		{
			ID:      "defaultLang",
			Type:    _select,
			Options: langs,
		},
		{
			ID:      "defaultCSS",
			Type:    _select,
			Options: themes,
		},
		{ID: "pyu"},
		{ID: "hats"},
		{
			ID:       "maxWidth",
			Type:     _number,
			Min:      1,
			Required: true,
		},
		{
			ID:       "maxHeight",
			Type:     _number,
			Min:      1,
			Required: true,
		},
		{
			ID:       "maxSize",
			Type:     _number,
			Min:      1,
			Required: true,
		},
		{
			ID:       "JPEGQuality",
			Type:     _number,
			Min:      1,
			Required: true,
			Max:      100,
		},
		{
			ID:       "PNGQuality",
			Type:     _number,
			Min:      1,
			Required: true,
		},
		{
			ID:   "FAQ",
			Type: _textarea,
			Rows: 5,
		},
		{
			ID:   "links",
			Type: _map,
		},
	},
}

// Specs of option inputs grouped by tab
var optionSpecs = [][]inputSpec{
	{
		{
			ID:      "lang",
			Type:    _select,
			Options: langs,
		},
		{ID: "imageHover"},
		{ID: "webmHover"},
		{ID: "notification"},
		{ID: "anonymise"},
		{ID: "relativeTime"},
		{ID: "alwaysLock"},
	},
	{
		{
			ID:      "inlineFit",
			Type:    _select,
			Options: []string{"none", "width", "screen"},
		},
		{ID: "hideThumbs"},
		{ID: "workModeToggle"},
		{ID: "autogif"},
		{ID: "spoilers"},
		{ID: "replyRight"},
		{
			ID:      "theme",
			Type:    _select,
			Options: themes,
		},
		{ID: "userBG"},
		{
			ID:   "userBGImage",
			Type: _image,
		},
	},
	{
		{ID: "google"},
		{ID: "iqdb"},
		{ID: "saucenao"},
		{ID: "desustorage"},
		{ID: "exhentai"},
	},
	{
		{ID: "nowPlaying"},
		{ID: "illyaDance"},
		{ID: "illyaDanceMute"},
		{ID: "horizontalPosting"},
	},
	{
		{
			ID:   "newPost",
			Type: _shortcut,
		},
		{
			ID:   "done",
			Type: _shortcut,
		},
		{
			ID:   "toggleSpoiler",
			Type: _shortcut,
		},
		{
			ID:   "expandAll",
			Type: _shortcut,
		},
		{
			ID:   "workMode",
			Type: _shortcut,
		},
	},
}
