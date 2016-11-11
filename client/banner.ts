// Handles all things related to the top banner

import { defer } from './defer'
import { banner as lang } from './lang'
import { write } from './render'
import View from "./view"
import Model from "./model"

// Stores the views of all BannerModal instances
export const bannerModals: { [key: string]: BannerModal } = {}

// View of the modal currently displayed, if any
let visible: BannerModal

// Highlight options button by fading out and in, if no options are set
function highlightBanner(name: string) {
	const key = name + "_seen"
	if (localStorage.getItem(key)) {
		return
	}

	const el = document.querySelector('#banner-' + name)
	write(() =>
		el.classList.add("blinking"))

	el.addEventListener("click", () => {
		el.classList.remove("blinking")
		localStorage.setItem(key, '1')
	})
}

defer(() =>
	["options", "FAQ", "identity", "account"]
		.forEach(highlightBanner))

// A modal element, that is positioned fixed right beneath the banner
export class BannerModal extends View<Model> {
	constructor(el: HTMLElement) {
		super({ el })
		bannerModals[this.id] = this

		// Add click listener to the toggle button of the modal in the banner
		document
			.querySelector('#banner-' + (this.id as string).split('-')[0])
			.addEventListener('click', () => this.toggle(), { capture: true })
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

	// Unhide the element. If the element has not been rendered yet, do it.
	show() {
		write(() =>
			this.el.style.display = 'block')
		visible = this
	}

	// Hide the element
	hide() {
		write(() =>
			this.el.style.display = 'none')
		visible = null
	}
}

// A view that supports switching between multiple tabs
export class TabbedModal extends BannerModal {
	constructor(el: HTMLElement) {
		super(el)
		this.onClick({
			'.tab-link': e =>
				this.switchTab(e),
		})
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
			for (let el of this.el.querySelectorAll(`.tab-cont > div`)) {
				if (el.getAttribute("data-id") !== id) {
					continue
				}
				el.classList.add("tab-sel")
			}
		})
	}
}

// Apply localized hover tooltips to banner links
function localizeTitles() {
	for (let id of ['feedback', 'FAQ', 'identity', 'options', 'account']) {
		setTitle('banner-' + id, id)
	}
	setTitle('sync', 'sync')
}

new BannerModal(document.getElementById("FAQ"))

defer(localizeTitles)

// Set the title of an element to a localized string
export function setTitle(id: string, langID: string) {
	write(() =>
		document.querySelector('#' + id)
			.setAttribute('title', lang[langID]))
}
