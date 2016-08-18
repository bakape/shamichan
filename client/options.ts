// User-set settings storage and change handling

import {emitChanges, ChangeEmitter} from './model'
import {OptionSpec, specs, optionType, OptionValue} from './options/specs'
import OptionsPanel from './options/view'
import {defer} from './defer'

// Delete legacy options localStorage entry, if any
localStorage.removeItem("options")

interface Options extends ChangeEmitter {
	lang: string
	inlineFit: string
	hideThumbs: boolean
	imageHover: boolean
	webmHover: boolean
	autogif: boolean
	spoilers: boolean
	notification: boolean
	anonymise: boolean
	relativeTime: boolean
	nowPlaying: boolean
	illyaDance: boolean
	illyaDanceMute: boolean
	horizontalPosting: boolean
	replyRight: boolean
	theme: string
	userBG: boolean
	lastN: number
	alwaysLock: boolean
	newPost: string
	toggleSpoiler: string
	textSpoiler: string
	done: string
	expandAll: string
	workMode: string
	workModeToggle: boolean
}

// Central options storage model
let options: Options
export default options = emitChanges({} as Options)

// All loaded option models
export const models: {[key: string]: OptionModel} = {}

// Controler for each individual option
class OptionModel {
	id: string
	spec: OptionSpec

	// Create new option model from template spec
	constructor(spec: OptionSpec) {
		// Condition for loading option. Optional.
		if (spec.noLoad) {
			return
		}
		this.spec = spec
		this.id = this.spec.id

		// No type = checkbox + default false
		if (!spec.type) {
			spec.type = optionType.checkbox
		}

		// Store option value in central stotage options Model
		const val = options[this.id] = this.get()
		options.onChange(this.id, val =>
			this.onChange(val))
		if (!spec.noExecOnStart) {
			this.execute(val)
		}
		models[this.id] = this
	}

	// Read value from localStorage
	read(): string {
		return localStorage.getItem(this.id) || ""
	}

	// Retrieve option value from storage and parse result. If none, return
	get(): OptionValue {
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
	onChange(val: OptionValue) {
		this.execute(val)
		this.set(val)
	}

	// Execute handler function, if any
	execute(val: OptionValue) {
		if (this.spec.exec) {
			this.spec.exec(val)
		}
	}

	// Write value to localStorage, if needed
	set(val: OptionValue) {
		if (val !== this.spec.default || this.read()) {
			localStorage.setItem(this.id, val.toString())
		}
	}

	// Perform value validation, if any. Othervise return true.
	validate(val: OptionValue): boolean {
		if (this.spec.validation) {
			return this.spec.validation(val)
		}
		return true
	}
}

// Intialize options system
export function initOptions() {
	// Populate option model collection and central model
	for (let spec of specs()) {
		new OptionModel(spec)
	}

	// Conditionally load and execute optional modules
	for (let opt of ["userBG", "nowPlaying"]) {
		if (options[opt]) {
			defer(() =>
				models[opt].execute(true))
		}
	}

	defer(() =>
		new OptionsPanel())
}
