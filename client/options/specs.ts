// Specs for individual option models

import { config } from '../state'
import lang from '../lang'
import { makeEl } from "../util"
import { render as renderBG } from "./background"
import initRadio from "./r-a-dio"

// Types of option models
export const enum optionType {
	checkbox, number, image, shortcut, menu, textarea,
}

// Full schema of the option interface
export type OptionSpec = {
	// Type of option. Determines storage and rendering method. Defaults to
	// 'checkbox', if omitted.
	type?: optionType

	// Default value. false, if omitted.
	default?: any

	// Function to execute on option change
	exec?: (val?: any) => void

	// Should the function not be executed on model population?
	noExecOnStart?: boolean

	// Function that validates the users input
	validation?: (val: any) => boolean
}

function renderBackground(_: boolean) {
	renderBG()
}

// Specifications of option behavior, where needed. Some properties defined as
// getters to prevent race with "state" module
export const specs: { [id: string]: OptionSpec } = {
	// Language selection
	lang: {
		type: optionType.menu,
		get default() {
			return config.defaultLang
		},
		noExecOnStart: true,
		exec(ln: string) {
			// Expire 10 years from now
			const t = new Date(new Date().getFullYear() + 10, 11)
			document.cookie = `lang=${ln};expires${t.toUTCString()};path=/`
			alert(lang.opts["langApplied"])
			location.reload()
		},
	},
	// Thumbnail inline expansion mode
	inlineFit: {
		type: optionType.menu,
		default: "width",
	},
	// Hide thumbnails
	hideThumbs: {},
	// Boss key toggle
	workModeToggle: {
		type: optionType.checkbox,
		default: false,
		exec: toggleHeadStyle("work-mode", ".image-banner{display: none;}"),
	},
	// Image hover expansion
	imageHover: {
		default: true,
	},
	// WebM hover expansion
	webmHover: {},
	// Animated GIF thumbnails
	autogif: {},
	// Enable thumbnail spoilers
	spoilers: {
		default: true,
	},
	// Desktop Notifications
	notification: {
		default: true,
		exec(enabled: boolean) {
			const can = enabled
				&& typeof Notification === "function"
				&& Notification.permission !== "granted"
			if (can) {
				Notification.requestPermission()
			}
		},
	},
	// Anonymise all poster names
	anonymise: {},
	// Expand post links inline
	postInlineExpand: {
		default: true,
		exec: toggleHeadStyle(
			"postInlineExpand",
			".hash-link{ display: inline; }"
		),
	},
	// Relative post timestamps
	relativeTime: {},
	// R/a/dio now playing banner
	nowPlaying: {
		noExecOnStart: true,
		exec: initRadio,
	},
	// Illya dance in the background
	illyaDance: {
		noExecOnStart: true,
		exec: renderBackground,
	},
	// Mute Illya dance
	illyaDanceMute: {
		noExecOnStart: true,
		exec: renderBackground,
	},
	// Tile posts horizontally too
	horizontalPosting: {
		exec: toggleHeadStyle(
			'horizontal',
			'article,aside{display:inline-block;}'
			+ '#thread-container{display:block;}'
		)
	},
	// Move [Reply] to the right side of the screen
	replyRight: {
		exec: toggleHeadStyle(
			'reply-at-right',
			'aside.posting{margin: -26px 0 2px auto;}'
		)
	},
	// Change theme
	theme: {
		type: optionType.menu,
		get default() {
			return config.defaultCSS
		},
		noExecOnStart: true,
		exec(theme: string) {
			if (!theme) {
				return
			}
			document
				.getElementById('theme-css')
				.setAttribute('href', `/assets/css/${theme}.css`)
		},
	},
	// Custom user-set background
	userBG: {
		noExecOnStart: true,
		exec: renderBackground,
	},
	// Upload field for the custom background image
	userBGImage: {
		type: optionType.image,
	},
	// User-set CSS rules
	customCSS: {
		type: optionType.textarea,
		exec(css: string) {
			let el = document.getElementById("custom-CSS-style")
			if (!el) {
				el = document.createElement("style")
				el.id = "custom-CSS-style"
				document.head.append(el)
			}
			el.innerHTML = css
		},
	},
	// Lock thread scrolling to bottom, when bottom in view, even when the
	// tab is hidden
	alwaysLock: {},
	// Image search link toggles
	google: {
		default: true,
		exec: toggleImageSearch("google"),
	},
	iqdb: {
		exec: toggleImageSearch("iqdb"),
	},
	saucenao: {
		default: true,
		exec: toggleImageSearch("saucenao"),
	},
	whatAnime: {
		exec: toggleImageSearch("whatAnime"),
	},
	desustorage: {
		exec: toggleImageSearch("desustorage"),
	},
	exhentai: {
		exec: toggleImageSearch("exhentai"),
	},
	// Shortcut keys
	newPost: {
		default: 78,
		type: optionType.shortcut,
	},
	done: {
		default: 83,
		type: optionType.shortcut,
	},
	toggleSpoiler: {
		default: 73,
		type: optionType.shortcut,
	},
	expandAll: {
		default: 69,
		type: optionType.shortcut,
	},
	workMode: {
		default: 66,
		type: optionType.shortcut,
	},
}

// Create a function that toggles the visibility of an image search link
function toggleImageSearch(engine: string): (toggle: boolean) => void {
	return toggleHeadStyle(engine, `.${engine}{display:initial;}`)
}

// Toggle an optional style element in the head
function toggleHeadStyle(
	name: string,
	css: string,
): (toggle: boolean) => void {
	return toggle => {
		const id = name + "-toggle"
		if (!document.getElementById(id)) {
			const html = `<style id="${id}">${css}</style>`
			document.head.append(makeEl(html))
		}

		// The disabled property only exists on elements in the DOM, so we do
		// another query
		(document.getElementById(id) as HTMLInputElement).disabled = !toggle
	}
}
