/*
 * This file is used by both the client to populate the Backbone models and the
 * server to render the actual options panel.
 */

var imports = require('./imports'),
	index = require('./index'),
	util = require('./util'),
	main = imports.main,
	$, Cookie, notMobile, state;
if (imports.isNode)
// TEMP: Will build separate templates and bundles for mobile eventually
	notMobile = true;
else {
	$ = require('jquery');
	Cookie = require('js-cookie');
	state = main.state;

	notMobile = !imports.main.isMobile;
}

const config = imports.config;
let hotConfig = imports.hotConfig;

/*
 * Full schema of the options model
 *
 * - id: Identifier of the option. Used for DOM element and localStorage tagging
 * - type: 'checkbox'/'number'/'image'/'shortCut'/array of options
 *	arrays become a selection list. Defaults to 'checkbox', if omitted.
 * - default: Default value. false, if omitted.
 * - tab: Index of the tab the option belong to.
 * - exec: Function to execute on option change.
 * - execOnStart: Boolean. Should the function be executed on model population?
 *	Defaults to true.
 * - load: Condition to display and execute the option. Defaults to true(always)
 * - validation: Function that validates the users input. Returns a boolean.
 *
 * Tooltips and lables are defined per language in `lang/`.
 * All arguments except for `id` and `tab` are optional.
 */

var opts = [
	/* LANGUAGE SELECTION */
	{
		id: 'lang',
		type: config.LANGS,
		tab: 0,
		default: config.DEFAULT_LANG,
		// True by default
		execOnStart: false,
		// Exec is not used on the server
		exec: function(type) {
			Cookie.set('lang', type);
			alert('Language settings applied. The page will now reload.');
			location.reload();
		}
	},
	/* INLINE EXPANSION */
	{
		id: 'inlinefit',
		type: ['none', 'full', 'width', 'height', 'both'],
		tab: 1,
		default: 'width'
	},
	/* THUMBNAIL OPTIONS */
	{
		id: 'thumbs',
		// Hardcoded to avoid circular dependancy on the server
		type: ['small', 'sharp', 'hide'],
		tab: 1,
		default: 'small',
		exec: function(type) {
			Cookie.set('thumb', type);
			main.oneeSama.thumbStyle = type;
		}
	},
	/* IMAGE HOVER EXPANSION */
	{
		id: 'imageHover',
		load: notMobile,
		tab: 0
	},
	{
		id: 'webmHover',
		load: notMobile,
		tab: 0
	},
	/* Autogif TOGGLE */
	{
		id: 'autogif',
		load: notMobile,
		tab: 1,
		exec: function(autogif) {
			Cookie.set('agif', autogif);
			main.oneeSama.autoGif = autogif;
		}
	},
	/* SPOILER TOGGLE */
	{
		id: 'spoilers',
		type: 'checkbox',
		tab: 1,
		default: true,
		exec: function(spoilertoggle) {
			Cookie.set('spoil', spoilertoggle);
			main.oneeSama.spoilToggle = spoilertoggle;
		}
	},
	/* LINKIFY TEXT URLS */
	{
		id: 'linkify',
		tab: 0,
		exec: function(toggle) {
			Cookie.set('linkify', toggle);
			main.oneeSama.eLinkify = toggle;
		}
	},
	/* DESKTOP NOTIFICATIONS */
	{
		id: 'notification',
		load: notMobile,
		tab: 0,
		exec: function(notifToggle) {
			if (notifToggle && (Notification.permission !== "granted"))
				Notification.requestPermission();
		}
	},
	/* ANONIMISE ALL POSTER NAMES */
	{
		id: 'anonymise',
		tab: 0
	},
	/* RELATIVE POST TIMESTAMPS */
	{
		id: 'relativeTime',
		tab: 0,
		exec: function(toggle) {
			main.oneeSama.rTime = toggle;
		}
	},
	/* R/A/DIO NOW PLAYING BANNER */
	{
		id: 'nowPlaying',
		load: notMobile && config.RADIO,
		type: 'checkbox',
		tab: 3,
		default: true,
		exec: function(toggle) {
			if (toggle)
				// Query the server for current stream info
				main.send([index.RADIO]);
			else
				main.request('banner:radio:clear');
		}
	}
];

/* IMAGE SEARCH LINK TOGGLE */
{
	const engines = ['google', 'iqdb', 'saucenao', 'foolz', 'exhentai'];
	for (let i = 0, l = engines.length; i < l; i++) {
		const engine = engines[i];
		opts.push({
			id: engine,
			// Use a custom internatiolisation function
			lang: 'imageSearch',
			tab: 2,
			type: 'checkbox',
			default: engine === 'google',
			exec: function(toggle) {
				var $style = $('#' + engine + 'Toggle');
				if (!$style.length) {
					$style = $('<style/>', {id: engine + 'Toggle'})
						.html('.' + engine + '{display:none;}')
						.appendTo('head');
				}
				$style.prop('disabled', toggle);
			}
		});
	}
}

opts.push(
	/* ILLYA DANCE */
	{
		id: 'illyaBGToggle',
		/*
		 The getters ensure there isn't any funny business with dependancy order
		 on the server;
		 */
		load: notMobile && hotConfig.ILLYA_DANCE,
		tab: 3
	},
	{
		id: 'illyaMuteToggle',
		load: notMobile && hotConfig.ILLYA_DANCE,
		tab: 3
	},
	/* HORIZONTAL POSTING */
	{
		id: 'horizontalPosting',
		tab: 3,
		exec: function(toggle) {
			var style = '<style id="horizontal">article,aside'
				+ '{display:inline-block;}</style>';
			if (toggle)
				$('head').append(style);
			else
				$('#horizontal').remove();
		}
	},
	/* REPLY AT RIGHT */
	{
		id: 'replyright',
		tab: 1,
		exec: function(r) {
			if (r)
				$('<style/>', {
					id: 'reply-at-right',
					text: 'section > aside { margin: -26px 0 2px auto; }'
				}).appendTo('head');
			else
				$('#reply-at-right').remove();
		}
	},
	/* THEMES */
	{
		id: 'theme',
		// Arrays will turn into selection boxes
		type: [
			'moe',
			'gar',
			'mawaru',
			'moon',
			'ashita',
			'console',
			'tea',
			'higan',
			'ocean',
			'rave',
			'tavern',
			'glass'
		],
		tab: 1,
		get default() {
			return hotConfig.DEFAULT_CSS;
		},
		exec: function(theme) {
			if (!theme)
				return;
			$('#theme').attr('href',
				`${config.MEDIA_URL}css/${theme}.css?v=${cssHash}`
			);
		}
	},
	/* CUSTOM USER-SET BACKGROUND */
	{
		id: 'userBG',
		load: notMobile,
		tab: 1
	},
	{
		id: 'userBGimage',
		load: notMobile,
		type: 'image',
		tab: 1,
		execOnStart: false,
		exec: function(upload) {
			main.request('background:store', upload);
		}
	},
	/* LAST N CONFIG */
	{
		id: 'lastn',
		type: 'number',
		tab: 0,
		validation: util.reasonable_last_n,
		default: hotConfig.THREAD_LAST_N,
		exec: function(n) {
			main.oneeSama.lastN = n;
			Cookie.set('lastn', n);
		}
	},
	/* KEEP THREAD LENGTH WITHIN LASTN */
	{
		id: 'postUnloading',
		tab: 0
	},
	/* LOCK TO BOTTOM EVEN WHEN DOCUMENT HIDDEN*/
	{
		id: 'alwaysLock',
		tab: 0
	}
);

/* SHORTCUT KEYS */
const shorts = [
	{id: 'new',					default: 78},
	{id: 'togglespoiler',	default: 73},
	{id: 'textSpoiler',		default: 68},
	{id: 'done',				default: 83},
	{id: 'expandAll',			default: 69}
];
for (let i = 0, lim = shorts.length; i < lim; i++) {
	let short = shorts[i];
	short.type = 'shortcut';
	short.tab = 4;
	opts.push(short);
}

module.exports = opts;
