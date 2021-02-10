// Specs for individual option models

import { config } from '../state'
import { makeEl, HTML, setCookie } from "../util"
import { render as renderBG } from "./background"
import { render as renderMascot } from "./mascot"
import { toggle as toggleNowPlaying } from "./nowPlaying"
import initTV from "./meguTV"
import options from "."

// Types of option models
export const enum optionType {
	checkbox, number, image, shortcut, menu, textarea, none, range,
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
}

function renderBackground(_: boolean) {
	renderBG()
}

// Specifications of option behavior, where needed. Some properties defined as
// getters to prevent race with "state" module
export const specs: { [id: string]: OptionSpec } = {
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
	// Volume of audio for music and video players
	audioVolume: {
		type: optionType.range,
		default: 100,
	},
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
			const req = enabled
				&& typeof Notification === "function"
				&& (Notification as any).permission !== "granted"
			if (req) {
				Notification.requestPermission()
			}
		},
	},
	// Thread watcher
	watchThreadsOnReply: {
		default: true,
	},
	// Anonymise all poster names
	anonymise: {},
	// Hide all deleted posts
	hideBinned: {},
	// Hide posts that linked to a hidden post
	hideRecursively: {},
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
	// Now playing banners
	horizontalNowPlaying: {
		default: true,
		exec: toggleHeadStyle(
			"horizontalNowPlaying",
			"#banner-center > div:not(.hidden) { display: inline; margin: 0.5em; }"
		),
	},
	radio: {
		exec: toggleNowPlaying(
			"https://r-a-d.io/",
			"https://r-a-d.io/api",
			({
				main: {
					dj: {
						djname: streamer,
					},
					listeners,
					np: song,
				}
			}) => ({ listeners, song, streamer }),
		),
	},
	eden: {
		exec: toggleNowPlaying(
			"https://www.edenofthewest.com/",
			"https://www.edenofthewest.com/api/live/nowplaying/eden_radio",
			({
				listeners: {
					current: listeners,
				},
				live: {
					streamer_name,
				},
				now_playing: {
					song: {
						text: song,
					},
				},
			}) => ({ listeners, song, streamer: streamer_name || "Eden Radio" }),
		),
	},
	shamiradio: {
		exec: toggleNowPlaying(
			"https://radio.shamik.ooo/",
			"https://radio.shamik.ooo/status-json.xsl",
			({ icestats: { source } }) => {
				const [fallback, main] = Array.isArray(source) ? source : [source]
				const { listeners, server_name: streamer, title: song = "oh dear, tags aren't set" }
					= main?.stream_start ? main : fallback
				return { listeners, song, streamer }
			},
		),
	},
	// User-specified video in the background
	bgVideo: {
		type: optionType.menu,
		default: "none",
		noExecOnStart: true,
		exec: renderBackground,
	},
	// Mute user-specified background video
	bgMute: {
		noExecOnStart: true,
		exec: renderBackground,
	},
	// Random video player
	meguTV: {
		noExecOnStart: true,
		exec: initTV,
	},
	// Tile posts horizontally too
	horizontalPosting: {
		exec: toggleHeadStyle(
			'horizontal',
			HTML`#thread-container {
				display:flex;
				flex-direction: row;
				flex-wrap: wrap;
				align-items: center;
			}`,
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
			// The server needs the theme cookie when nagivating from
			// a third party website, so we set SameSite=Lax.
			setCookie("theme", theme, 365 * 10, "lax")
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
	// Mascot in the corner
	mascot: {
		noExecOnStart: true,
		exec: renderMascot,
	},
	mascotImage: {
		type: optionType.image,
	},
	// User-set CSS rules
	customCSSToggle: {
		noExecOnStart: true,
		exec(on: boolean) {
			let el = document
				.getElementById("custom-CSS-style") as HTMLStyleElement
			if (!el) {
				el = document.createElement("style")
				el.id = "custom-CSS-style"
				document.head.append(el)
				// The disabled property only exists on elements in the DOM,
				// so we do another query
				el = document
					.getElementById("custom-CSS-style") as HTMLStyleElement
			}
			el.innerHTML = options.customCSS;
			(el as any).disabled = !on
		},
	},
	customCSS: {
		type: optionType.textarea,
	},
	// Lock thread scrolling to bottom, when bottom in view, even when the
	// tab is hidden
	alwaysLock: {},
	// Image search link toggles
	google: {
		default: true,
		exec: toggleImageSearch("google"),
	},
	yandex: {
		exec: toggleImageSearch("yandex"),
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
	meguTVShortcut: {
		default: 84,
		type: optionType.shortcut,
	},
	galleryModeToggle: {
		type: optionType.none,
		exec: toggleHeadStyle(
			"gallery",
			`#threads article:not(.media),
			.fileinfo,
			blockquote,
			.backlinks,
			header > :not(nav),
			header .quote,
			header .mod-checkbox {
				display: none;
			}
			header, figcaption {
				display: inline-block;
			}
			#thread-container, article:not(.reply-form) {
				display: inline-table;
			}
			.post-container {
				display: flex;
				min-width: initial;
			}
			figure {
				margin: 0;
				margin-left: auto;
				margin-right: auto;
			}
			figcaption {
				text-align: center;
			}
			article {
				padding: 0.5em;
				width: fit-content;
			}
			a[download], header nav {
				font-size: 0;
			}
			header nav a:first-child::before {
				content: "# ";
				font-size: 15px;
			}
			a[download]::before {
				content: " 🡇";
				font-size: 15px;
			}`,
		)
	},
	galleryMode: {
		default: 71,
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
		(document.getElementById(id) as any).disabled = !toggle
	}
}
