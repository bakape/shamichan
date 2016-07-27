import {on, OnOptions} from './util'
import Model from './model'
import {write} from './render'

export interface ViewAttrs {
	el?: Element
	model?: Model
	tag?: string
	class?: string
	id?: string
}

// Generic view class, that all over view classes extend
export default class View<M> {
	model: M
	el: Element
	id: string|number

	// Creates a new View and binds it to the target model, id any. If no
	// element suplied, creates a new one from the attributes.
	constructor({el, model, tag, class: cls, id}: ViewAttrs) {
		if (model) {
			this.model = model as any
		}
		if (!el) {
			this.el = document.createElement(tag || 'div')
			if (id) {
				this.el.setAttribute('id', id)
				this.id = id
			}
			if (cls) {
				this.el.setAttribute('class', cls)
			}
		} else {
			this.el = el
			const id = el.getAttribute('id')
			if (id) {
				this.id = id
			}
		}
	}

	// Remove the from the DOM without causing a redraw
	remove() {
		write(() =>
			this.el.remove())
	}

	// Add  optionally selector-specific event listeners to the view
	on(type: string, fn: EventListener, opts?: OnOptions) {
		on(this.el, type, fn, opts)
	}

	// Shorthand for adding multiple click event listeners as an object.
	// We use those the most, so nice to have. Also prevents default behavior
	// from triggering.
	onClick(events: {[selector: string]: EventListener}) {
		for (let selector in events) {
			this.on('click', events[selector], {selector, capture: true})
		}
	}
}
