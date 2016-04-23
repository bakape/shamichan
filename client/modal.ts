/*
  Various minor windows and the base view for all modals
 */

import {default as View, ViewAttrs} from './view'

// Modal elements, that float above other content
export class Modal extends View {
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

	// TODO: Add close button and unify modal structure

}
