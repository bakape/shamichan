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
		document.body.append(this.el)
		this.render()
	}

    // Placeholder method for overriding
    render() {}

	// TODO: Add close button and unify modal structure

}

// A modal element, that is positioned fixed right beneath the banner
export class BannerModal extends Modal {
	constructor(args: ViewAttrs) {
		args.cls = 'bmodal'
		super(args)
	}
}
