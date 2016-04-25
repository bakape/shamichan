// Handles all things related to the top banner

import {config} from './state'
import {defer} from './defer'
import {Modal} from './modal'
import {ViewAttrs} from './view'
import {banner as lang} from './lang'
import {write, read} from './render'

// Highlight options button by fading out and in, if no options are set
function highlightBanner() {
	if (localStorage.getItem('optionsSeen')) {
		return
	}
	const el = document.query('#banner-options')
	el.style.opacity = '1'
	let out = true,
		clicked: boolean
	el.addEventListener("click", () => {
		clicked = true
		localStorage.setItem('optionsSeen', '1')
	})
	tick()

	function tick() {
		// Stop
		if (clicked) {
			el.style.opacity = '1'
			return
		}

		el.style.opacity = (+el.style.opacity + (out ? -0.02 : 0.02)).toString()
		const now = +el.style.opacity

		// Reverse direction
		if ((out && now <= 0) || (!out && now >= 1)) {
			out = !out
		}
		requestAnimationFrame(tick)
	}
}

defer(highlightBanner)

// Stores the views of all BannerModal instances
export const bannerModals: {[key: string]: BannerModal} = {}

// View of the modal currently displayed, if any
let visible: BannerModal

// A modal element, that is positioned fixed right beneath the banner
export class BannerModal extends Modal {
	constructor(args: ViewAttrs) {
		super(args)
		bannerModals[this.id] = this
		read(() =>
			document
				.query('#banner-' + (this.id as string).split('-')[0])
				.addEventListener('click', () => this.toggle()))
	}

	// Show the element, if hidden, hide - if shown. Hide already visible
	// banner modal, if any.
	toggle() {
		if (visible) {
			const old = visible
			visible.hide()
			if (old !== this) {
				this.show()
			}
		} else {
			this.show()
		}
	}

	// Unhide the element
	private show() {
		write(() => this.el.style.display = 'inline-table')
		visible = this
	}

	// Hide the element
	private hide() {
		write(() => this.el.style.display = 'none')
		visible = null
	}
}

// Frequently asked question and other information modal
class FAQPanel extends BannerModal {
	constructor() {
		super({el: document.query('#FAQ-panel')})
		this.render()
	}

	render() {
		let html = '<ul>'
		for (let line of config.FAQ) {
			html += `<li>${line}</line>`
		}
		html += `</ul>`
		write(() => this.el.innerHTML = html)
	}
}

defer(() => new FAQPanel())

// Name and email input pannel
class IdentityPanel extends BannerModal {
	constructor() {
		super({el: document.query('#identity-panel')})
		write(() => this.render())
	}

	render() {
		this.el.query('label[for=name]').textContent = lang.name
		this.el.query('label[for=email]').textContent = lang.email
	}
}

defer(() => new IdentityPanel())

// Apply localised hover tooltips to banner links
function localiseTitles() {
	for (let id of ['feedback', 'FAQ', 'identity', 'options']) {
		setTitle('banner-' + id, id)
	}
	for (let id of ['sync', 'onlineCount']) {
		setTitle(id, id)
	}
}

defer(() =>	write(localiseTitles))

function setTitle(id: string, langID: string) {
	document.query('#' + id).setAttribute('title', lang[langID])
}
