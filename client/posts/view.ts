import View, {ViewAttrs} from '../view'
import {Post} from './models'

// Base post view class
export default class PostView<M extends Post<any>> extends View<M> {
	constructor(attrs: ViewAttrs) {
		super(attrs)
		this.model.view = this
	}

	// Remove the element from the DOM and detach from its model, allowing the
	// PostView instance to be garbage collected.
	remove() {
		delete this.model.view
		delete this.model
		super.remove()
	}
}
