/**
 * Various minor windows and the base view for all modals
 */

import View from './view'

// Modal elements, that float above other content
export class Modal extends View {
	constructor(args) {
		// Child classes must always pass an `attrs` object, in the arguments
		// object
		const addClass = 'modal glass'
		if (args.class) {
			args.class += ' ' + addClass
		} else {
			args.class = addClass
		}
		super(args)
		this.render()
		document.body.append(this.el)
	}

	// TODO: Add close button and unify modal structure

}

// A modal element, that is positioned fixed right beneath the banner
export class BannerModal extends Modal {
	constructor(args) {
		args.class = 'bmodal'
		super(args)
	}
}
