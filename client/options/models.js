import {extend} from '../../vendor/underscore'
import options from '../options'
import opts from './opts'

// All loaded option models
const optionModels = {}
export default optionModels

/**
 * Coontroler for each individual option
 */
class OptionModel {
	/**
	 * Create new option model from template model
	 * @param {Object} model
	 */
    constructor(model) {
		// Condition for loading option. Optional.
		if (model.load !== undefined && !model.load) {
			return
		}
		extend(this, model)

		// No type = checkbox + default false
		if (!this.type) {
		    this.type = 'checkbox'
		}

		// Store option value in central stotage options Backbone model
		const val = options.attrs[this.id] = this.get()
		options.onChange(this.id, val =>
			this.onChange(val))
		if (this.execOnStart !== false) {
		    this.execute(val)
		}
		optionModels[this.id] = this
    }

	/**
	 * Read value from localStorage
	 * @returns {string}
	 */
	read() {
	    return localStorage.getItem(this.id)
	}

	/**
	 * Retrieve option value from storage and parse result. If none, return
	 * default.
	 * @returns {string|bool|int}
	 */
	get() {
		const stored = this.read()
	    if (!stored) {
	        return this.default
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
			return this.default
		}
	}

	/**
	 * Handler to be executed on field change in central options storage model
	 * @param {*} val
	 */
	onChange(val) {
	    this.execute(val)
		this.set(val)
	}

	/**
	 * Execute handler function, if any
	 * @param {*} val
	 */
	execute(val) {
	    if (this.exec) {
	        this.exec(val)
	    }
	}

	/**
	 * Write value to localStorage, if needed
	 * @param {*} val
	 */
	set(val) {
	    if (val !== this.default || this.read()) {
	        localStorage.setItem(this.id, val)
	    }
	}

	/**
	 * Perform value validation, if any. Othervise return true.
	 * @param {*} val
	 * @returns {bool}
	 */
	validate(val) {
	    if (this.validation) {
	        return this.validation(val)
	    }
		return true
	}
}

// Create an option model for each object in the array
for (let spec of opts) {
	new OptionModel(spec)
}
