/**
 * Various minor windows and the base view for all modals
 */
/* @flow */

import View from './view'


// Modal elements, that float above other content
export class Modal extends View {
	constructor(args :Object) {
		// Child classes must always pass an `attrs` object, in the arguments
		// object
		const addClass = 'modal glass'
		if (args.class) {
			args.class += ' ' + addClass
		} else {
			args.class = addClass
		}
		super(args)
	}

	// TODO: Add close button and unify modal structure

}

// A modal element, that is positioned fixed right beneath the banner
export class BannerModal extends Modal {
	constructor(args :Object) {
		args.class = 'bmodal'
		super(args)
	}
}
