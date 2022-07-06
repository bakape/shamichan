// Specifications for various input elements

package templates

import (
	"github.com/bakape/meguca/common"
)

// NOTE: After adding inputSpec structs with new ID fields, be sure to add the
// description to at least `lang/en_GB/server.json.forms`. Then run
// `scripts/migrate_lang.js` to insert temporary placeholders into any language
// packs missing translations.

// Reused in multiple places
var (
	repeatPasswordSpec = inputSpec{
		ID:           "repeat",
		Type:         _password,
		MaxLength:    common.MaxLenPassword,
		NoID:         true,
		Required:     true,
		Autocomplete: "new-password",
	}
	sageSpec         = inputSpec{ID: "sage"}
	staffTitleSpec   = inputSpec{ID: "staffTitle"}
	defaultThemeSpec = inputSpec{
		ID:      "defaultCSS",
		Type:    _select,
		Options: common.Themes,
	}
)

var specs = map[string][]inputSpec{
	"identity": {
		sageSpec,
		{
			ID:           "name",
			Type:         _string,
			MaxLength:    common.MaxLenName,
			Autocomplete: "off",
		},
	},
	"noscriptPostCreation": {
		{
			ID:           "name",
			Type:         _string,
			MaxLength:    common.MaxLenName,
			Placeholder:  true,
			Autocomplete: "off",
			NoID:         true,
		},
		inputSpec{
			ID:          "body",
			Type:        _textarea,
			Rows:        5,
			MaxLength:   common.MaxLenBody,
			Placeholder: true,
			NoID:        true,
		},
	},
	"login": {
		{
			ID:           "id",
			Type:         _string,
			MaxLength:    common.MaxLenUserID,
			NoID:         true,
			Required:     true,
			Autocomplete: "username",
		},
		{
			ID:           "password",
			Type:         _password,
			MaxLength:    common.MaxLenPassword,
			NoID:         true,
			Required:     true,
			Autocomplete: "current-password",
		},
	},
	"register": {
		{
			ID:           "id",
			Type:         _string,
			MaxLength:    common.MaxLenUserID,
			NoID:         true,
			Required:     true,
			Autocomplete: "off",
		},
		{
			ID:           "password",
			Type:         _password,
			MaxLength:    common.MaxLenPassword,
			NoID:         true,
			Required:     true,
			Autocomplete: "new-password",
		},
		repeatPasswordSpec,
	},
	"changePassword": {
		{
			ID:           "oldPassword",
			Type:         _password,
			MaxLength:    common.MaxLenPassword,
			NoID:         true,
			Required:     true,
			Autocomplete: "current-password",
		},
		{
			ID:           "newPassword",
			Type:         _password,
			MaxLength:    common.MaxLenPassword,
			NoID:         true,
			Required:     true,
			Autocomplete: "new-password",
		},
		repeatPasswordSpec,
	},
	"configureBoard": {
		{ID: "readOnly"},
		{ID: "textOnly"},
		{ID: "forcedAnon"},
		{ID: "randomNameHours"},
		{ID: "disableRobots"},
		{ID: "flags"},
		{ID: "NSFW"},
		{ID: "rbText"},
		{Type: _hr},
		{ID: "pyu"},
		{
			ID:        "title",
			Type:      _string,
			MaxLength: common.MaxLenBoardTitle,
		},
		{
			ID:        "notice",
			Type:      _textarea,
			Rows:      5,
			MaxLength: common.MaxLenNotice,
		},
		{
			ID:        "rules",
			Type:      _textarea,
			Rows:      5,
			MaxLength: common.MaxLenRules,
		},
		defaultThemeSpec,
		{
			ID:        "eightball",
			Type:      _array,
			MaxLength: common.MaxLenEightball,
		},
	},
	"createBoard": {
		{
			ID:        "boardName",
			Type:      _string,
			Required:  true,
			Pattern:   "^[a-z0-9]{1,10}$",
			MaxLength: common.MaxLenBoardID,
		},
		{
			ID:        "boardTitle",
			Type:      _string,
			Required:  true,
			MaxLength: common.MaxLenBoardTitle,
		},
	},
	"configureServer": {
		{ID: "mature"},
		{ID: "hideNSFW"},
		{ID: "disableUserBoards"},
		{ID: "globalDisableRobots"},
		{Type: _hr},
		{ID: "pruneThreads"},
		{
			ID:       "threadExpiryMin",
			Type:     _number,
			Min:      1,
			Required: true,
		},
		{
			ID:       "threadExpiryMax",
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
		{Type: _hr},
		{
			ID:   "rootURL",
			Type: _string,
		},
		{
			ID:   "imageRootOverride",
			Type: _string,
		},
		{Type: _hr},
		{
			ID:   "salt",
			Type: _string,
		},
		{ID: "captcha"},
		{
			ID:   "captchaTags",
			Type: _array,
		},
		{
			ID:   "overrideCaptchaTags",
			Type: _map,
		},
		{
			ID:       "charScore",
			Type:     _number,
			Min:      0,
			Required: true,
		},
		{
			ID:       "postCreationScore",
			Type:     _number,
			Min:      0,
			Required: true,
		},
		{
			ID:       "imageScore",
			Type:     _number,
			Min:      0,
			Required: true,
		},
		{
			ID:       "sessionExpiry",
			Type:     _number,
			Min:      1,
			Required: true,
		},
		{Type: _hr},
		{ID: "emailErr"},
		{
			ID:           "emailErrMail",
			Type:         _string,
			MaxLength:    common.MaxLenUserID,
			Required:     true,
			Autocomplete: "off",
		},
		{
			ID:           "emailErrPass",
			Type:         _password,
			MaxLength:    common.MaxLenPassword,
			Required:     true,
			Autocomplete: "off",
		},
		{
			ID:       "emailErrSub",
			Type:     _string,
			Required: true,
		},
		{
			ID:       "emailErrPort",
			Type:     _number,
			Min:      0,
			Required: true,
		},
		{Type: _hr},
		{
			ID:   "feedbackEmail",
			Type: _string,
		},
		{
			ID:      "defaultLang",
			Type:    _select,
			Options: common.Langs,
		},
		defaultThemeSpec,
		{Type: _hr},
		{ID: "JPEGThumbnails"},
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
		{Type: _hr},
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
var optionSpecs = [...][]inputSpec{
	{
		{ID: "imageHover"},
		{ID: "webmHover"},
		{
			ID:   "audioVolume",
			Type: _range,
			Min:  0,
			Max:  100,
		},
		{Type: _hr},
		{ID: "notification"},
		{ID: "watchThreadsOnReply"},
		{Type: _hr},
		{ID: "anonymise"},
		{ID: "hideBinned"},
		{ID: "hideRecursively"},
		{Type: _hr},
		{ID: "postInlineExpand"},
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
		{Type: _hr},
		{ID: "replyRight"},
		{ID: "horizontalPosting"},
		{
			ID:      "theme",
			Type:    _select,
			Options: common.Themes,
		},
		{Type: _hr},
		{ID: "userBG"},
		{
			ID:   "userBGImage",
			Type: _image,
		},
		{ID: "mascot"},
		{
			ID:   "mascotImage",
			Type: _image,
		},
		{ID: "customCSSToggle"},
		{
			ID:   "customCSS",
			Type: _textarea,
			Rows: 3,
		},
	},
	{
		{ID: "google"},
		{ID: "yandex"},
		{ID: "iqdb"},
		{ID: "saucenao"},
		{ID: "tracemoe"},
		{ID: "desuarchive"},
		{ID: "exhentai"},
	},
	{
		{ID: "horizontalNowPlaying"},
		{ID: "radio"},
		{ID: "eden"},
		{ID: "shamiradio"},
		{ID: "shamiradio2"},
		{Type: _hr},
		{
			ID:      "bgVideo",
			Type:    _select,
			Options: []string{"none"},
		},
		{ID: "bgMute"},
		{ID: "meguTV"},
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
		{
			ID:   "galleryMode",
			Type: _shortcut,
		},
		{
			ID:   "meguTVShortcut",
			Type: _shortcut,
		},
	},
}
