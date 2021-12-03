// User-set settings storage and change handling

import { OptionSpec, specs, optionType } from './specs'
import initBackground from "./background"
import initLoops from "./loop"
import initMascot from "./mascot"
import { trigger, emitChanges, ChangeEmitter, hook } from "../util"
import { bgVideos } from "../state"

export { store as storeBackground } from "./background"
export { store as storeMascot } from "./mascot"
export * from "./specs"
export { getPostName } from "./nowPlaying"
export { persistMessages } from "./meguTV"

interface Options extends ChangeEmitter {
	hideThumbs: boolean
	imageHover: boolean
	webmHover: boolean
	autogif: boolean
	spoilers: boolean
	notification: boolean
	watchThreadsOnReply: boolean
	anonymise: boolean
	postInlineExpand: boolean
	relativeTime: boolean
	meguTV: boolean
	horizontalNowPlaying: boolean
	radio: boolean
	eden: boolean
	shamiradio: boolean
	bgVideo: string
	bgMute: boolean
	horizontalPosting: boolean
	hideBinned: boolean
	hideRecursively: boolean
	replyRight: boolean
	galleryModeToggle: boolean
	workModeToggle: boolean
	userBG: boolean
	customCSSToggle: boolean
	mascot: boolean
	alwaysLock: boolean
	newPost: number
	toggleSpoiler: number
	done: number
	expandAll: number
	workMode: number
	meguTVShortcut: number
	audioVolume: number
	inlineFit: string
	theme: string
	customCSS: string
}

// Central options storage model
let options = {} as Options
// Need to define all properties ahead of time for the ES5 Proxy polyfill to
// work
for (let k in specs) {
	options[k] = undefined
}
export default options = emitChanges({} as Options)

// Provide workaround path to access options. Some core modules would cause
// circular imports otherwise.
hook("getOptions", () =>
	options)

// All loaded option models
export const models: { [key: string]: OptionModel } = {}

// Controller for each individual option
class OptionModel {
	public id: string
	public spec: OptionSpec

	// Create new option model from template spec
	constructor(id: string, spec: OptionSpec) {
		this.spec = spec
		this.id = id

		// No type = checkbox + default false
		if (!spec.type) {
			spec.type = optionType.checkbox
		}

		// Store option value in central storage options Model
		const val = options[this.id] = this.get()
		options.onChange(this.id, val =>
			this.onChange(val))
		if (!spec.noExecOnStart) {
			this.execute(val)
		}
		models[this.id] = this
	}

	// Read value from localStorage
	private read(): string {
		return localStorage.getItem(this.id) || ""
	}

	// Retrieve option value from storage and parse result. If none, return
	public get(): any {
		const stored = this.read()
		if (!stored) {
			return this.spec.default
		} else {
			if (stored === 'false') {
				return false
			}
			if (stored === "true") {
				return true
			}
			const num = parseInt(stored, 10)
			if (num || num === 0) {
				return num
			}
			return stored
		}
	}

	// Handler to be executed on field change in central options storage model
	private onChange(val: any) {
		this.execute(val)
		this.set(val)
	}

	// Execute handler function, if any
	public execute(val: any) {
		if (this.spec.exec) {
			this.spec.exec(val)
		}
	}

	// Write value to localStorage, if needed
	public set(val: any) {
		if (this.id === "meguTV") {
			return;
		}
		if (val !== this.spec.default || this.read()) {
			localStorage.setItem(this.id, val.toString())
		}
		trigger("renderOptionValue", this.id, this.spec.type, val)
	}
}

function migrateOption(key: string, fn: (value: string) => void) {
	const value = localStorage.getItem(key)
	if (value !== null) {
		fn(value)
		localStorage.removeItem(key)
	}
}

export function initOptions() {
	// Delete legacy options localStorage entry, if any
	localStorage.removeItem("options")

	// Migrate old options
	migrateOption("nowPlaying", value => {
		switch (value) {
			case "true":
			case "r/a/dio":
				localStorage.setItem("radio", "true")
				break
			case "eden":
				localStorage.setItem("eden", "true")
				break
			case "both":
				localStorage.setItem("radio", "true")
				localStorage.setItem("eden", "true")
				break
			default:
				const flags = parseInt(value, 10)
				for (const [i, key] of ["radio", "eden", "shamiradio"].entries()) {
					if (flags & 1 << i) {
						localStorage.setItem(key, "true")
					}
				}
		}
	})
	migrateOption("whatAnime", value => localStorage.setItem("tracemoe", value))
	migrateOption("desustorage", value => localStorage.setItem("desuarchive", value))

	// Populate option model collection and central model
	for (let id in specs) {
		new OptionModel(id, specs[id])
	}

	// Manually change bgVideo select as it is dynamically generated
	const bgElement = document.getElementById("bgVideo")
	bgElement.innerHTML = ""

	for (let val of bgVideos) {
		const opt = document.createElement("option")
		opt.value = val
		opt.innerText = val
		bgElement.append(opt);
	}

	// Conditionally load and execute optional modules
	for (let opt of ["userBG", "bgVideo", "mascot", "customCSSToggle", "meguTV"]) {
		if (options[opt]) {
			models[opt].execute(true)
		}
	}

	// Change the applied custom CSS on CSS change
	options.onChange("customCSS", () => {
		if (options.customCSSToggle) {
			models["customCSSToggle"].execute(true)
		}
	})

	initBackground()
	initMascot()
	initLoops()
}

// Return, if user can be notified with desktop notifications
export function canNotify(): boolean {
	return options.notification
		&& typeof Notification === "function"
		&& (Notification as any).permission === "granted";
}

// Returns, if images can be shown on the page
export function canShowImages(): boolean {
	return !options.hideThumbs && !options.workModeToggle;
}

// Construct common base for all notification options
export function notificationOpts(): NotificationOptions {
	const re: NotificationOptions = {};
	if (canShowImages()) {
		re.icon = re.badge = "/assets/notification-icon.png";
	}
	return {
		vibrate: 500,
	};
}
