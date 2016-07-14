import View, {ViewAttrs} from '../view'
import {Post} from './models'

// Base post view class
export default class PostView<M extends Post<any>> extends View<M> {
	constructor(attrs: ViewAttrs) {
		super(attrs)
		this.model.attach(this)
	}

	// Remove the element from the DOM and detach from its model, allowing the
	// PostView instance to be garbage collected.
	remove() {
		this.model.detach(this)
		delete this.model
		super.remove()
	}
}
