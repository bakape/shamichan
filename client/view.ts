import {on, once, onceAll} from './util'
import Model from './model'

export type ViewAttrs = {
	el?: Element
	model?: Model
	tag?: string
	cls?: string
	id?: string
}

// Generic view class, that all over view classes extend
export default class View {
	model: Model
	el: Element

	// Creates a new View and binds it to the target model. If none, creates a
	// blank model. If no element suplied, creates a new one from tags. Sets
	// class and id, if supplied.
	constructor({el, model, tag, cls, id}: ViewAttrs) {
		this.model = model || new Model()
		this.el = el || document.createElement(tag || 'div')
		if (id) {
			this.el.setAttribute('id', id)
		}
		if (cls) {
			this.el.setAttribute('class', cls)
		}
		this.model.attach(this)
	}

	// Remove the element from the DOM and detach from its model, allowing the
	// View instance to be garbage collected.
	remove() {
		this.el.remove()
		this.model.detach(this)
		delete this.model
	}

	// Add selector-specific event listeners to the view
	on(type: string, selector: string, fn: EventListener) {
		on(this.el, type, selector, fn)
	}

	// Shorthand for adding multiple click event listeners as an object.
	// We use those the most, so nice to have.
	onClick(events: {[selector: string]: EventListener}) {
		for (let selector in events) {
			this.on('click', selector, events[selector])
		}
	}

	// Add event listener to view's element, whithout filtering by selector
	onAll(type: string, fn: EventListener) {
		this.el.addEventListener(type, fn)
	}

	// Add selector-specific event listener, that will execute only once on a
	// specific target
	once(type: string, selector: string, fn: EventListener) {
		once(this.el, type, selector, fn)
	}

	// Add event listener, that will execute only once
	onceAll(type: string, fn: EventListener) {
		onceAll(this.el, type, fn)
	}
}
