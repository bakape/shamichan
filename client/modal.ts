import {default as View, ViewAttrs} from './view'

// Modal elements, that float above other content
export default class Modal<M> extends View<M> {
	constructor(args: ViewAttrs) {
		let cls = 'modal glass'
		if (args.class) {
			cls += " " + args.class
		}
		args.class = cls
		super(args)
	}
}
