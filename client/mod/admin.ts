// Server administration controls, such as global server settings
import View from '../view'

type ServerConfigs = {
	prune: boolean
	radio: boolean
	hats: boolean
	illyaDance: boolean
	maxWidth: number
	maxHeight: number
	maxThreads: number
	maxBump: number
	JPEGQuality: number
	PNGQuality: number
	threadCooldown: number
	maxSubjectLength: number
	maxSize: number
	defaultLang: string
	frontPage: string
	origin: string
	defaultCSS: string
	salt: string
	excludeRegex: string
	feedbackEmail: string
	FAQ: string
	langs: string[]
	links: string[][]
	spoilers: number[]
	sessionExpiry: number
}

class ConfigPanel extends View {
	$parent: Element

	constructor(parent: Element) {
		super({})
		this.$parent = parent
	}
}
