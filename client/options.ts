/*
 User-set settings storage and change handling
*/

import Model from './model'
import {extend} from './util'
import {OptionSpec, specs, optionType, OptionValue} from './options/specs'
import OptionsPanel from './options/view'
import {defer} from './defer'

// Delete legacy options localStorage entry, if any
localStorage.removeItem("options")

// Central options storage model
const options = new Model()
export default options

// All loaded option models
export const models: {[key: string]: OptionModel} = {}

// Option model IDs
export type OptionID =
	'lang' | 'inlineFit' | 'thumbs' | 'imageHover' | 'webmHover' | 'autogif'
	| 'spoilers' | 'notification' | 'anonymise' | 'relativeTime' | 'nowPlaying'
	| 'illyaDance' | 'illyaDanceMute' | 'horizontalPosting' | 'replyRight'
	| 'theme' | 'userBG' | 'userBGImage' | 'lastN' | 'alwaysLock' | 'newPost'
	| 'toggleSpoiler' | 'textSpoiler' | 'done' | 'expandAll' |'workMode'
	| 'export' | 'import' | 'hidden' | 'workModeToggle' | 'google' | 'iqdb'
	| 'saucenao' | 'desustorage' | 'exhentai'

// Controler for each individual option
class OptionModel {
	id: OptionID
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
		const val = options.attrs[this.id] = this.get()
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
	private onChange(val: OptionValue) {
		this.execute(val)
		this.set(val)
	}

	// Execute handler function, if any
	private execute(val: OptionValue) {
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

// Populate option model collection and central model
for (let spec of specs) {
	new OptionModel(spec)
}

defer(() => new OptionsPanel())
