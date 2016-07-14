import {default as View, ViewAttrs} from './view'
import Model from './model'

// Modal elements, that float above other content
export default class Modal<M> extends View<M> {
	constructor(args: ViewAttrs) {
		// Child classes must always pass a ViewAttrs object
		const addClass = 'modal glass'
		if (args.cls) {
			args.cls += ' ' + addClass
		} else {
			args.cls = addClass
		}
		super(args)
	}
}
