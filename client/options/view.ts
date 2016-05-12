import {BannerModal} from '../banner'
import renderContents from './render'
import {models, default as options} from '../options'
import {optionType} from './specs'
import {each, find, loadModule} from '../util'
import {opts as lang} from '../lang'
import {write, read} from '../render'

// View of the options panel
export default class OptionsPanel extends BannerModal {
	$hidden: Element

	constructor() {
		super({el: document.query('#options-panel')})
		this.render()
		this.onClick({
			'.tab_link': e => this.switchTab(e),
			'#export': () => this.exportConfigs(),
			'#import': e => this.importConfigs(e),
			'#hidden': () => this.clearHidden()
		})
		this.onAll('change', e => this.applyChange(e))
	}

	// Render the contents of the options panel and insert it into the DOM
	render() {
		const html = renderContents()
		write(() => {
			this.el.innerHTML = html
			this.assignValues()
		})
		read(() => this.$hidden = this.el.query('#hidden'))

		// TODO: Hidden posts count rendering
		// events.reply('hide:render', this.renderHidden, this)
	}

	// Assign loaded option settings to the respective elements in the options
	// panel
	assignValues() {
		for (let id in models) {
			const model = models[id],
				el = this.el.query('#' + id),
				val = model.get()
			switch (model.spec.type) {
			case optionType.checkbox:
				el.checked = val as boolean
				break
			case optionType.number:
			case optionType.menu:
				el.value = val
				break
			case optionType.shortcut:
				el.value = String.fromCharCode(val as number).toUpperCase()
				break
			}
			// 'image' type simply falls through, as those don't need to be set
		}
	}

	// Propagate options panel changes through
	// options-panel -> options -> OptionModel
	applyChange(event: Event) {
		const el = event.target as Element,
			id = el.getAttribute('id'),
			model = models[id]
		let val: boolean|string|number
		switch (model.spec.type) {
		case optionType.checkbox:
			val = el.checked
			break
		case optionType.number:
			val = parseInt(el.value)
			break
		case optionType.menu:
			val = el.value
			break
		case optionType.shortcut:
			val = el.value.toUpperCase().charCodeAt(0)
			break
		case optionType.image:
			// Not recorded. Extracted directly by the background handler
			loadModule('background').then(module =>
				module.store((event as any).target.files[0]))
			return
		}

		if (!model.validate(val)) {
			el.value = ''
		} else {
			options.set(id, val)
		}
	}

	// Switch to a tab, when clicking the tab butt
	switchTab(event: Event) {
		write(() => {
			const el = event.target as Element

			// Deselect previous tab
			each<Element>(this.el.children, el =>
				el.query('.tab_sel').classList.remove('tab_sel'))

			// Select the new one
			el.classList.add('tab_sel')
			find<Element>(this.el.lastChild.children, li =>
				li.classList.contains(el.getAttribute('data-content'))
			)
				.classList.add('tab_sel')
		})
	}

	// Dump options to JSON file and upload to user
	exportConfigs() {
		const a = document.getElementById('export')
		const blob = new Blob([JSON.stringify(localStorage)], {
			type: 'octet/stream'
		})
		a.setAttribute('href', window.URL.createObjectURL(blob))
		a.setAttribute('download', 'meguca-config.json')
	}

	// Import options from uploaded JSON file
	importConfigs(event: Event) {
		// Proxy to hidden file input
		const el = document.query('#importSettings')
		el.click()
		this.onceAll('change', () => {
			const reader = new FileReader()
			reader.readAsText(el.files[0])
			reader.onload = event => {
				event as FileReaderOnloadEvent

				// In case of curruption
				let json: {[key: string]: string}
				try {
					json = JSON.parse(event.target.result)
				}
				catch(err) {
					alert(lang.importConfig.corrupt)
					return
				}
				localStorage.clear()
				for (let key in json) {
					localStorage.setItem(key, json[key])
				}
				alert(lang.importConfig.done)
				location.reload()
			}
		})
	}

	// Render Hiden posts counter
	renderHidden(count: number) {
		write(() => {
			const el = this.$hidden
			el.textContent = el.textContent.replace(/\d+$/, count.toString())
		})
	}

	// Clear displayed hidden post counter
	clearHidden() {

		// TODO: Fix  after post hiding ported
		// main.request('hide:clear')

		this.renderHidden(0)
	}
}
