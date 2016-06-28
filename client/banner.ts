// Handles all things related to the top banner

import {config} from './state'
import {defer} from './defer'
import {Modal} from './modal'
import {ViewAttrs} from './view'
import {banner as lang} from './lang'
import {write, read} from './render'
import {setLabel, find} from './util'

// Highlight options button by fading out and in, if no options are set
function highlightBanner(name: string) {
	const key = name + "_seen"
	if (localStorage.getItem(key)) {
		return
	}

	const el = document.querySelector('#banner-' + name)
	el.style.opacity = '1'
	let out = true,
		clicked: boolean
	el.addEventListener("click", () => {
		clicked = true
		localStorage.setItem(key, '1')
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

defer(() => ["options", "FAQ", "identity", "account"].forEach(highlightBanner))

// Stores the views of all BannerModal instances
export const bannerModals: {[key: string]: BannerModal} = {}

// View of the modal currently displayed, if any
let visible: BannerModal

// A modal element, that is positioned fixed right beneath the banner
export class BannerModal extends Modal {
	constructor(args: ViewAttrs) {
		super(args)
		bannerModals[this.id] = this

		// Add click listener to the toggle button of the modal in the banner
		document
			.querySelector('#banner-' + (this.id as string).split('-')[0])
			.addEventListener('click', () => this.toggle(), {capture: true})
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

// A view that supports switching between multiple tabs
export class TabbedModal extends BannerModal {
	constructor(args: ViewAttrs) {
		super(args)
		this.onClick({'.tab-link': e => this.switchTab(e)})
	}

	// Switch to a tab, when clicking the tab butt
	switchTab(event: Event) {
		write(() => {
			const el = event.target as Element

			// Deselect previous tab
			for (let selected of this.el.querySelectorAll('.tab-sel')) {
				selected.classList.remove('tab-sel')
			}

			// Select the new one
			el.classList.add('tab-sel')
			const id = el.getAttribute('data-id')
			find<Element>(this.el.querySelector(".tab-cont").children, li =>
				li.getAttribute('data-id') === id
			)
				.classList.add('tab-sel')
		})
	}
}

// Frequently asked questions and other information modal
defer(() =>
	new BannerModal({
		el: document.querySelector('#FAQ-panel')
	}))

// Name and email input pannel
class IdentityPanel extends BannerModal {
	constructor() {
		super({el: document.querySelector('#identity-panel')})
		write(() => this.render())
	}

	render() {
		for (let name of ["name", "email"]) {
			setLabel(this.el, name, lang[name])
		}
	}
}

defer(() => new IdentityPanel())

// Apply localised hover tooltips to banner links
function localiseTitles() {
	for (let id of ['feedback', 'FAQ', 'identity', 'options', 'account']) {
		setTitle('banner-' + id, id)
	}
	setTitle('sync', 'sync')
}

defer(() =>	write(localiseTitles))

function setTitle(id: string, langID: string) {
	document.querySelector('#' + id).setAttribute('title', lang[langID])
}
