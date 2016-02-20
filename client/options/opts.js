import * as Cookie from 'js-cookie'
import {thumbStyles, resonableLastN, parseEl} from '../util'
import {config, isMobile} from '../state'
import {langApplied} from 'lang'

// TODO: Send function

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

const notMobile = !isMobile

const opts = [
	// LANGUAGE SELECTION
	{
		id: 'lang',
		type: config.lang.enabled,
		tab: 0,
		default: config.lang.default,
		execOnStart: false,
		exec(type) {
			alert(langApplied)
			location.reload()
		}
	},
	// INLINE EXPANSION
	{
		id: 'inlinefit',
		type: ['none', 'full', 'width', 'height', 'both'],
		tab: 1,
		default: 'width'
	},
	// THUMBNAIL MODE
	{
		id: 'thumbs',
		type: thumbStyles,
		tab: 1,
		default: 'small'
	},
	// IMAGE HOVER EXPANSION
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
	// Autogif TOGGLE
	{
		id: 'autogif',
		load: notMobile,
		tab: 1
	},
	// SPOILER TOGGLE
	{
		id: 'spoilers',
		tab: 1,
		default: true
	},
	// LINKIFY TEXT URLS
	{
		id: 'linkify',
		tab: 0,
		default: true
	},
	// DESKTOP NOTIFICATIONS
	{
		id: 'notification',
		load: notMobile,
		tab: 0,
		exec(notifToggle) {
			if (notifToggle && (Notification.permission !== "granted"))
				Notification.requestPermission()
		}
	},
	// ANONIMISE ALL POSTER NAMES
	{
		id: 'anonymise',
		tab: 0
	},
	// RELATIVE POST TIMESTAMPS
	{
		id: 'relativeTime',
		tab: 0,
		default: true
	},
	// R/A/DIO NOW PLAYING BANNER
	{
		id: 'nowPlaying',
		load: notMobile && config.radio,
		tab: 3,
		default: true,
		exec(toggle) {
			if (toggle) {
				// Query the server for current stream info
				send({type: 'radio'})
			} else {

				// TODO: System.import().then()
				//events.request('banner:radio:clear');
			}
		}
	},
	// ILLYA DANCE
	{
		id: 'illyaBGToggle',
		load: notMobile && config.illyaDance,
		tab: 3
	},
	{
		id: 'illyaMuteToggle',
		load: notMobile && config.illyaDance,
		tab: 3
	},
	// HORIZONTAL POSTING
	{
		id: 'horizontalPosting',
		tab: 3,
		exec: toggleHeadStyle(
			'horizontal',
			'article,aside{display:inline-block;}'
		)
	},
	// REPLY AT RIGHT
	{
		id: 'replyright',
		tab: 1,
		exec: toggleHeadStyle(
			'reply-at-right',
			'section>aside{margin: -26px 0 2px auto;}'
		)
	},
	// THEMES
	{
		id: 'theme',
		type: [
			'moe', 'gar', 'mawaru', 'moon', 'ashita', 'console', 'tea',
			'higan', 'ocean', 'rave', 'tavern', 'glass'
		],
		tab: 1,
		default: config.defaultCSS,
		execOnStart: false,
		exec(theme) {
			if (!theme) {
				return
			}
			document.getElementById('theme')
				.setAttribute(
					'href',
					`/ass/css/${theme}.css`
				)
		}
	},
	// CUSTOM USER-SET BACKGROUND
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
			// TODO: System.import().then()
			//events.request('background:store', upload)
		}
	},
	// LAST N CONFIG
	{
		id: 'lastn',
		type: 'number',
		tab: 0,
		validation: resonableLastN,
		default: 100
	},
	// KEEP THREAD LENGTH WITHIN LASTN
	/*
	 Disabled, until dependancy features are implemnted (see issue #280)
	{
		id: 'postUnloading',
		tab: 0
	},*/
	// LOCK TO BOTTOM EVEN WHEN DOCUMENT HIDDEN
	{
		id: 'alwaysLock',
		tab: 0
	}
]

// IMAGE SEARCH LINK TOGGLE
for (let engine of ['google', 'iqdb', 'saucenao', 'desustorage', 'exhentai']) {
	opts.push({
		id: engine,
		// Use a custom internatiolisation function
		lang: 'imageSearch',
		tab: 2,
		default: engine === 'google',
		exec: toggleHeadStyle(engine + 'Toggle', `.${engine}{display:initial;}`)
	})
}

// SHORTCUT KEYS
const shorts = [
	{id: 'new', default: 78},
	{id: 'togglespoiler', default: 73},
	{id: 'textSpoiler', default: 68},
	{id: 'done', default: 83},
	{id: 'expandAll', default: 69},
	{id: 'workMode', default: 66}
]
for (let short of shorts) {
	short.type = 'shortcut'
	short.tab = 4
	short.load = notMobile
	opts.push(short)
}

// Create a function to append and toggle a style element in <head>
function toggleHeadStyle(id, css) {
	return toggle => {
		if (!document.getElementById(id)) {
			document.head.appendChild(
				parseEl(`<style id="${id}">${css}</style>`)
			)
		}

		// The disabled property only exists on elements in the DOM, so we do
		// another query
		document.getElementById(id).disabled = !toggle
	}
}

// Exports the generated option model templates
export default opts
