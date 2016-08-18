// Specs for individual option models

import {config, isMobile} from '../state'
import {opts as lang} from '../lang'
import {loadModule} from '../util'

// Types of option models
export const enum optionType {checkbox, number, image, shortcut, menu}

// Options panel display tabs
export const enum tabs {general, style, imagesearch, fun, shortcuts}

// Can't  use enums for ones below, because they persist to localStorage

// Thumbnail expansion modes
export const thumbExpansions = ['none', 'width', 'screen']

// Available themes. Change this, when adding any new ones.
export const themes = [
	'moe', 'gar', 'mawaru', 'moon', 'ashita', 'console', 'tea', 'higan',
	'ocean', 'rave', 'glass', 'material',
]

// Available language packs. Change this, when adding any new ones.
export const langs = ["en_GB", "es_ES", "pt_BR", "tr_TR"]

export type OptionValue =  boolean|string|number

// Full schema of the option interface
export type OptionSpec = {
	// Identifier of the option. Used for DOM element and localStorage tagging
	id: string

	// Type of option. Determines storage and rendering method. Defaults to
	// 'checkbox', if omitted.
	type?: optionType

	// Index of the tab the option belong to
	tab: tabs

	// Items to place in a <select> list
	list?: string[]

	// Default value. false, if omitted.
	default?: OptionValue

	// Function to execute on option change
	exec?: (val?: OptionValue) => void

	// Should the function not be executed on model population?
	noExecOnStart?: boolean

	// Condition, when not to display and execute the option
	noLoad?: boolean

	// Function that validates the users input
	validation?: (val: OptionValue) => boolean

	// Don't show the option to the user in the option's panel
	hidden?: boolean
}

// Option position in the options panel is defined by order in the
// array. A function, so we can ensure it is not created before state.ts is
// loaded.
export const specs = (): OptionSpec[] => [
	// Language selection
	{
		id: 'lang',
		type: optionType.menu,
		list: langs,
		tab: tabs.general,
		default: config.defaultLang,
		noExecOnStart: true,
		exec() {
			alert(lang.langApplied)
			location.reload()
		},
	},

	// Thumbnail inline expansion mode
	{
		id: 'inlineFit',
		type: optionType.menu,
		list: thumbExpansions,
		tab: tabs.style,
		default: 'width'
	},

	// Hide thumbnails, until explicitly revealed
	{
		id: 'hideThumbs',
		tab: tabs.style,
	},

	// // Image hover expansion
	// {
	// 	id: 'imageHover',
	// 	default: true,
	// 	noLoad: isMobile,
	// 	tab: tabs.general
	// },
	// // WebM hover expansion
	// {
	// 	id: 'webmHover',
	// 	noLoad: isMobile,
	// 	tab: tabs.general
	// },

	// Animated GIF thumbnails
	{
		id: 'autogif',
		noLoad: isMobile,
		tab: tabs.style,
	},

	// Enable thumnail spoilers
	{
		id: 'spoilers',
		tab: tabs.style,
		default: true,
	},

	// // Desktop Notifications
	// {
	// 	id: 'notification',
	// 	tab: tabs.general,
	// 	exec(toggle) {
	// 		if (toggle && Notification.permission !== "granted") {
	// 			Notification.requestPermission()
	// 		}
	// 	}
	// },

	// Anonymise all poster names
	{
		id: 'anonymise',
		tab: tabs.general,
	},

	// Relative post timestamps
	{
		id: 'relativeTime',
		tab: tabs.general,
		default: false,
	},

	// R/a/dio now playing banner
	{
		id: 'nowPlaying',
		noLoad: isMobile || !config.radio,
		tab: tabs.fun,
		default: true,
		noExecOnStart: true,
		exec() {
			loadModule("r-a-dio")
		},
	},

	// Illya dance in the background
	{
		id: 'illyaDance',
		noLoad: isMobile || !config.illyaDance,
		tab: tabs.fun,
	},
	// Mute Illya dance
	{
		id: 'illyaDanceMute',
		noLoad: isMobile || !config.illyaDance,
		tab: tabs.fun,
	},

	// // Tile posts horizontally too
	// {
	// 	id: 'horizontalPosting',
	// 	tab: tabs.fun,
	// 	exec: toggleHeadStyle(
	// 		'horizontal',
	// 		'article,aside{display:inline-block;}'
	// 	)
	// },
	// // Move [Reply] to the right side of the screen
	// {
	// 	id: 'replyRight',
	// 	tab: tabs.style,
	// 	exec: toggleHeadStyle(
	// 		'reply-at-right',
	// 		'section>aside{margin: -26px 0 2px auto;}'
	// 	)
	// },

	// Change theme
	{
		id: 'theme',
		type: optionType.menu,
		list: themes,
		tab: tabs.style,
		default: config.defaultCSS,
		noExecOnStart: true,
		exec(theme) {
			if (!theme) {
				return
			}
			document.getElementById('theme').setAttribute(
				'href',
				`/assets/css/${theme}.css`
			)
		},
	},

	// Custom user-set background
	{
		id: 'userBG',
		noLoad: isMobile,
		tab: tabs.style,
		noExecOnStart: true,
		exec() {
			loadModule('background').then(m =>
				m.render())
		}
	},
	// Upload field for the custom background image
	{
		id: 'userBGImage',
		noLoad: isMobile,
		type: optionType.image,
		tab: tabs.style
	},

	// Last N posts to display in a thread, if viewing in Last N mode
	{
		id: 'lastN',
		type: optionType.number,
		tab: tabs.general,
		validation(n: number) {
			return Number.isInteger(n) && n <= 500
		},
		default: 100,
	},

	// KEEP THREAD LENGTH WITHIN LASTN
	// Disabled, until dependancy features are implemnted (see issue #280)
	// {
	// 	id: 'postUnloading',
	// 	tab: 0
	// },
	// Lock thread scrolling to bottom, when bottom in view, even when the tab
	// is hidden
	// {
	// 	id: 'alwaysLock',
	// 	tab: tabs.general
	// }
]

// TODO: Selective rendering logic
//
// Image search link toggle
// for (let engine of ['google', 'iqdb', 'saucenao', 'desustorage', 'exhentai']) {
// 	opts.push({
// 		id: engine,
// 		// Use a custom internatiolisation function
// 		lang: 'imageSearch',
// 		tab: 2,
// 		default: engine === 'google',
// 		exec: toggleHeadStyle(engine + 'Toggle', `.${engine}{display:initial;}`)
// 	})
// }

// TODO: Reimplement shortcu keys
// // SHORTCUT KEYS
// const shorts: any = [
// 	{id: 'newPost', default: 78},
// 	{id: 'toggleSpoiler', default: 73},
// 	{id: 'textSpoiler', default: 68},
// 	{id: 'done', default: 83},
// 	{id: 'expandAll', default: 69},
// 	{id: 'workMode', default: 66}
// ]
// for (let short of shorts) {
// 	short.type = optionType.shortcut
// 	short.tab = tabs.shortcuts
// 	short.noLoad = isMobile
// 	specs.push(short)
// }
//
// // Create a function to append and toggle a style element in <head>
// function toggleHeadStyle(id: string, css: string): (toggle: boolean) => void {
// 	return toggle => {
// 		if (!document.getElementById(id)) {
// 			document.head.append(makeEl(`<style id="${id}">${css}</style>`))
// 		}
//
// 		// The disabled property only exists on elements in the DOM, so we do
// 		// another query
// 		(document.getElementById(id) as HTMLInputElement).disabled = !toggle
// 	}
// }
