import Model from './model'
import {extend} from 'underscore'

// Generic view class, that all over view classes extend
export default class View {
	// Creates a new View and binds it to the target model. If none, creates a
	// blank model. If no element suplied, creates a new one from tags. Sets
	// some other default variables.
	constructor(args) {
		extend(this, args)
		if (!this.model) {
			this.model = new Model()
		}
		this.model.attach(this)

		// Create element, if none
		if (!this.el) {
			const el = this.el = document.createElement(this.tag || 'div')

			// Set element attributes, if any
			for (let key of ['id', 'class']) {
				if (key in args) {
					el.setAttribute(key, args[key])
				}
			}
		}
	}

	// Remove the element from the DOM and detach from its model, allowing the
	// View instance to be garbage collected.
	remove() {
		this.el.remove()
		this.model.detach(this)
		delete this.model
	}

	// Add selector-specific event listeners to the view
	on(type, selector, fn) {
		this.el.addEventListener(type, event => {
			if (event.target.matches && event.target.matches(selector)) {
				fn(event)
			}
		})
	}

	// Shorthand for adding multiple click event listeners as an object.
	// We use those the most, so nice to have.
	onClick(events) {
		for (let selector in events) {
			this.on('click', selector, events[selector])
		}
	}

	// Add event listener to view's element, whithout filtering by selector
	onAll(type, fn) {
		this.el.addEventListener(type, fn)
	}

	// Add selector-specific event listener, that will execute only once
	once(type, selector, fn) {
		this.el.addEventListener(type, event => {
			if (event.target.matches && event.target.matches(selector)) {
				fn(event)
				this.el.removeEventListener(type, fn)
			}
		})
	}

	// Add event listener, that will execute only once
	onceAll(type, fn) {
		this.el.addEventListener(type, event => {
			fn(event)
			this.el.removeEventListener(type, fn)
		})
	}
}
