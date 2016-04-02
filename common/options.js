/*
 * This file is used by both the client to populate the Backbone models and the
 * server to render the actual options panel.
 */

const imports = require('./imports'),
	index = require('./index'),
	util = require('./util'),
	{parseHTML} = util,
	{config, hotConfig, main} = imports;
if (!imports.isNode)
	var {Cookie, etc, oneeSama, state} = main;

/*
 * Full schema of the option interface
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
 * - hidden: If true this option won't be shown to the user. Defaults to false
 *
 * Tooltips and lables are defined per language in `lang/`.
 * All arguments except for `id` and `tab` are optional.
 */

// Generate either a desktop or mobile set of options
module.exports = function(isMobile) {
	const notMobile = !isMobile;
	let opts = [
		/* LANGUAGE SELECTION */
		{
			id: 'lang',
			type: config.LANGS,
			tab: 0,
			default: config.DEFAULT_LANG,
			// True by default
			execOnStart: false,
			// Exec is not used on the server
			exec(type) {
				Cookie.set('lang', type);
				alert(main.lang.langApplied);
				location.reload(true);
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
			exec(type) {
				Cookie.set('thumb', type);
				oneeSama.thumbStyle = type;
			}
		},
		/* IMAGE HOVER EXPANSION */
		{
			id: 'imageHover',
			default: true,
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
			exec(autogif) {
				Cookie.set('agif', autogif);
				oneeSama.autoGif = autogif;
			}
		},
		/* SPOILER TOGGLE */
		{
			id: 'spoilers',
			tab: 1,
			default: true,
			exec(spoilertoggle) {
				Cookie.set('spoil', spoilertoggle);
				oneeSama.spoilToggle = spoilertoggle;
			}
		},
		/* LINKIFY TEXT URLS */
		{
			id: 'linkify',
			tab: 0,
			exec(toggle) {
				Cookie.set('linkify', toggle);
				oneeSama.eLinkify = toggle;
			}
		},
		/* DESKTOP NOTIFICATIONS */
		{
			id: 'notification',
			load: notMobile,
			tab: 0,
			exec(notifToggle) {
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
			default: true,
			exec(toggle) {
				oneeSama.rTime = toggle;
			}
		},
		/* R/A/DIO NOW PLAYING BANNER */
		{
			id: 'nowPlaying',
			load: notMobile && config.RADIO,
			tab: 3,
			default: true,
			exec(toggle) {
				if (toggle)
					// Query the server for current stream info
					main.send([index.RADIO]);
				else
					main.request('banner:radio:clear');
			}
		}
	];

	/* IMAGE SEARCH LINK TOGGLE */
	for (let engine of ['google', 'iqdb', 'saucenao', 'desustorage', 'exhentai']) {
		opts.push({
			id: engine,
			// Use a custom internatiolisation function
			lang: 'imageSearch',
			tab: 2,
			default: engine === 'google',
			exec: toggleHeadStyle(engine + 'Toggle',
				`.${engine}{display:initial;}`)
		});
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
			exec: toggleHeadStyle('horizontal',
				'article,aside{display:inline-block;}')
		},
		/* REPLY AT RIGHT */
		{
			id: 'replyright',
			tab: 1,
			exec: toggleHeadStyle('reply-at-right',
				'section>aside{margin: -26px 0 2px auto;}')
		},
		/* THEMES */
		{
			id: 'theme',
			// Arrays will turn into selection boxes
			type: [
				'moe', 'gar', 'mawaru', 'moon', 'ashita', 'console', 'tea',
				'higan', 'ocean', 'rave', 'tavern', 'glass', 'material'
			],
			tab: 1,
			default: hotConfig.DEFAULT_CSS,
			exec(theme) {
				if (!theme)
					return;
				document.getElementById('theme').setAttribute('href',
					`${config.MEDIA_URL}css/${theme}.css?v=${main.cssHash}`);
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
			exec(upload) {
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
			exec(n) {
				oneeSama.lastN = n;
				Cookie.set('lastn', n);
			}
		},
		/* KEEP THREAD LENGTH WITHIN LASTN */
		/*
		 Disabled, until dependancy features are implemnted (see issue #280)
		{
			id: 'postUnloading',
			tab: 0
		},*/
		/* LOCK TO BOTTOM EVEN WHEN DOCUMENT HIDDEN*/
		{
			id: 'alwaysLock',
			tab: 0
		}
	);

	/* SHORTCUT KEYS */
	const shorts = [
		{id: 'new', default: 78},
		{id: 'togglespoiler', default: 73},
		{id: 'textSpoiler', default: 68},
		{id: 'done', default: 83},
		{id: 'expandAll', default: 69},
		{id: 'workMode', default: 66}
	];
	for (let short of shorts) {
		short.type = 'shortcut';
		short.tab = 4;
		short.load = notMobile;
		opts.push(short);
	}

	return opts;
};

// Create a function to append and toggle a style element in <head>
function toggleHeadStyle(id, css) {
	return function (toggle) {
		if (!document.getElementById(id)) {
			const el = etc.parseDOM(`<style id="${id}">${css}</style>`)[0];
			document.head.appendChild(el);
		}

		// The disabled property only exists on elements in the DOM, so we do
		// another query
		document.getElementById(id).disabled = !toggle;
	}
}
