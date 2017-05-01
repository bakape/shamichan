// Specifications for various input elements

package templates

import (
	"meguca/common"
)

// NOTE: After adding inputSpec structs with new ID fields, be sure to add the
// description to at least `lang/en_GB/server.json.forms`. Then run
// `scripts/migrate_lang.js` to insert temporary placeholders into any language
// packs missing translations.

var specs = map[string][]inputSpec{
	"identity": {
		{ID: "staffTitle"},
		{ID: "live"},
		{ID: "sage"},
		{
			ID:           "name",
			Type:         _string,
			MaxLength:    common.MaxLenName,
			Autocomplete: "off",
		},
	},
	"noscriptPostCreation": {
		{ID: "sage"},
		{
			ID:           "name",
			Type:         _string,
			MaxLength:    common.MaxLenName,
			Placeholder:  true,
			Autocomplete: "off",
		},
		{
			ID:        "body",
			Type:      _textarea,
			Rows:      5,
			MaxLength: common.MaxLenBody,
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
		{
			ID:           "repeat",
			Type:         _password,
			MaxLength:    common.MaxLenPassword,
			NoID:         true,
			Required:     true,
			Autocomplete: "new-password",
		},
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
		{
			ID:           "repeat",
			Type:         _password,
			MaxLength:    common.MaxLenPassword,
			NoID:         true,
			Required:     true,
			Autocomplete: "new-password",
		},
	},
	"configureBoard": {
		{ID: "readOnly"},
		{ID: "textOnly"},
		{ID: "forcedAnon"},
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
			Pattern:   "^[a-z0-9]{1,3}$",
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
		{ID: "disableUserBoards"},
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
			ID:   "rootURL",
			Type: _string,
		},
		{
			ID:   "imageRootOverride",
			Type: _string,
		},
		{
			ID:   "salt",
			Type: _string,
		},
		{ID: "captcha"},
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
			Options: common.Langs,
		},
		{
			ID:      "defaultCSS",
			Type:    _select,
			Options: common.Themes,
		},
		{ID: "pyu"},
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
		{
			ID:      "lang",
			Type:    _select,
			Options: common.Langs,
		},
		{ID: "imageHover"},
		{ID: "webmHover"},
		{ID: "notification"},
		{ID: "anonymise"},
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
		{ID: "replyRight"},
		{ID: "horizontalPosting"},
		{
			ID:      "theme",
			Type:    _select,
			Options: common.Themes,
		},
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
		{
			ID:   "customCSS",
			Type: _textarea,
			Rows: 3,
		},
	},
	{
		{ID: "google"},
		{ID: "iqdb"},
		{ID: "saucenao"},
		{ID: "whatAnime"},
		{ID: "desustorage"},
		{ID: "exhentai"},
	},
	{
		{ID: "nowPlaying"},
		{ID: "illyaDance"},
		{ID: "illyaDanceMute"},
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
	},
}
