// Specs for individual option models

import { config, isMobile } from '../state'
import { opts as lang } from '../lang'
import { loadModule, makeEl } from '../util'
import { write } from "../render"

// Types of option models
export const enum optionType { checkbox, number, image, shortcut, menu }

// Options panel display tabs
export const enum tabs { general, style, imagesearch, fun, shortcuts }

// Can't  use enums for ones below, because they persist to localStorage

// Thumbnail expansion modes
export const thumbExpansions = ['none', 'width', 'screen']

// Available themes. Change this, when adding any new ones.
export const themes = [
	'moe', 'gar', 'mawaru', 'moon', 'ashita', 'console', 'tea', 'higan',
	'ocean', 'rave', 'glass',
]

// Available language packs. Change this, when adding any new ones.
export const langs = ["en_GB", "es_ES", "pt_BR", "sk_SK", "tr_TR", 'uk_UA']

export type OptionValue = boolean | string | number

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

// Same handler fot toggling Illya dance, adn user backgrounds
const renderBackground = () =>
	loadModule('background').then(m =>
		m.render())

// Option position in the options panel is defined by order in the
// array. A function, so we can ensure it is not created before state.ts is
// loaded.
export const specs = (): OptionSpec[] => {
	const opts = [
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

		{
			id: "workModeToggle",
			tab: tabs.style,
			exec: toggleHeadStyle("work-mode", ".image-banner{display: none;}"),
		},

		// Image hover expansion
		{
			id: 'imageHover',
			default: true,
			noLoad: isMobile,
			tab: tabs.general
		},
		// WebM hover expansion
		{
			id: 'webmHover',
			noLoad: isMobile,
			tab: tabs.general
		},

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
			noExecOnStart: true,
			exec: renderBackground,
		},
		// Mute Illya dance
		{
			id: 'illyaDanceMute',
			noLoad: isMobile || !config.illyaDance,
			tab: tabs.fun,
			noExecOnStart: true,
			exec: renderBackground,
		},

		// Tile posts horizontally too
		{
			id: 'horizontalPosting',
			tab: tabs.fun,
			exec: toggleHeadStyle(
				'horizontal',
				'article,aside{display:inline-block;}'
				+ '#thread-container{display:block;}'
			)
		},
		// Move [Reply] to the right side of the screen
		{
			id: 'replyRight',
			tab: tabs.style,
			exec: toggleHeadStyle(
				'reply-at-right',
				'aside.posting{margin: -26px 0 2px auto;}'
			)
		},

		// Change theme
		{
			id: 'theme',
			type: optionType.menu,
			list: themes,
			tab: tabs.style,
			default: config.defaultCSS,
			noExecOnStart: true,
			exec(theme: string) {
				if (!theme) {
					return
				}
				document
					.getElementById('theme')
					.setAttribute('href', `/assets/css/${theme}.css`)
			},
		},

		// Custom user-set background
		{
			id: 'userBG',
			noLoad: isMobile,
			tab: tabs.style,
			noExecOnStart: true,
			exec: renderBackground,
		},
		// Upload field for the custom background image
		{
			id: 'userBGImage',
			noLoad: isMobile,
			type: optionType.image,
			tab: tabs.style
		},

		// KEEP THREAD LENGTH WITHIN LASTN
		// Disabled, until dependancy features are implemnted (see issue #280)
		// {
		// 	id: 'postUnloading',
		// 	tab: 0
		// },

		// Lock thread scrolling to bottom, when bottom in view, even when the
		// tab is hidden
		{
			id: 'alwaysLock',
			tab: tabs.general
		},
	]

	// Image search link toggles
	const engines = ['google', 'iqdb', 'saucenao', 'desustorage', 'exhentai']
	for (let engine of engines) {
		opts.push({
			id: engine,
			tab: 2,
			default: engine === 'google',
			exec: toggleImageSearch(engine)
		})
	}

	// Shortcut keys
	const keySpecs: any[] = [
		{ id: 'newPost', default: 78 },
		{ id: 'done', default: 83 },
		{ id: 'toggleSpoiler', default: 73 },
		{ id: 'expandAll', default: 69 },
		{ id: 'workMode', default: 66 },
	]

	for (let spec of keySpecs as OptionSpec[]) {
		spec.type = optionType.shortcut
		spec.tab = tabs.shortcuts
		spec.noLoad = isMobile
		opts.push(spec)
	}

	return opts
}

// Create a function that toggles the visibility of an image search link
function toggleImageSearch(engine: string): (toggle: boolean) => void {
	return toggleHeadStyle(engine, `.${engine}{display:initial;}`)
}

// Toogle an optional style element in the head
function toggleHeadStyle(name: string, css: string): (toggle: boolean) => void {
	return toggle => {
		const id = name + "-toggle"
		if (!document.getElementById(id)) {
			const html = `<style id="${id}">${css}</style>`
			write(() =>
				document.head.append(makeEl(html)))
		}

		// The disabled property only exists on elements in the DOM, so we do
		// another query
		write(() =>
			(document.getElementById(id) as HTMLInputElement)
				.disabled = !toggle)
	}
}
