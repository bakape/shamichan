import {on, OnOptions} from './util'
import Model from './model'
import {write} from './render'

export type ViewAttrs = {
	el?: Element
	model?: Model
	tag?: string
	cls?: string
	id?: string
	parent?: View<Model>
}

// Generic view class, that all over view classes extend
export default class View<M extends Model> {
	model: M
	el: Element
	id: string|number
	parent: View<Model>

	// Creates a new View and binds it to the target model. If none, creates a
	// blank model. If no element suplied, creates a new one from tags. Sets
	// class and id, if supplied.
	constructor({el, model, tag, cls, id, parent}: ViewAttrs) {
		this.model = model || new Model() as any
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
		if (parent) {
			this.parent = parent
		}
		this.model.attach(this)
	}

	// Remove the element from the DOM and detach from its model, allowing the
	// View instance to be garbage collected.
	remove() {
		this.model.detach(this)
		delete this.model
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
