import { TabbedModal } from '../banner'
import renderContents from './render'
import { models, default as options } from '../options'
import { optionType } from './specs'
import { loadModule, load } from '../util'
import { opts as lang } from '../lang'
import { write, read } from '../render'
import { clearHidden } from "../posts/hide"
import { hidden } from "../state"

// Only instance of the options panel
export let panel: OptionsPanel

// View of the options panel
export default class OptionsPanel extends TabbedModal {
	hidden: Element
	import: HTMLInputElement

	constructor() {
		super({ id: 'options' })
		panel = this
		this.onClick({
			'#export': () =>
				this.exportConfigs(),
			'#import': e =>
				this.importConfigs(e),
			'#hidden': clearHidden,
		})
		this.on('change', e =>
			this.applyChange(e))

	}

	// Render the contents of the options panel and insert it into the DOM
	render() {
		this.lazyRender(renderContents())
		write(() =>
			this.assignValues())
		read(() => {
			this.hidden = this.el.querySelector('#hidden')
			this.renderHidden(hidden.size)
			this.import =
				this.el.querySelector("#importSettings") as HTMLInputElement
		})
	}

	// Assign loaded option settings to the respective elements in the options
	// panel
	assignValues() {
		for (let id in models) {
			const model = models[id],
				val = model.get()
			this.assignValue(id, model.spec.type, val)
		}
	}

	// Assign a single option value. Called on changes to the options externally
	// not from the options panel
	assignValue(id: string, type: optionType, val: any) {
		const el = this.el.querySelector('#' + id) as HTMLInputElement

		// Panel not rendered yet
		if (!el) {
			return
		}

		switch (type) {
			case optionType.checkbox:
				el.checked = val as boolean
				break
			case optionType.number:
			case optionType.menu:
				el.value = val as string
				break
			case optionType.shortcut:
				el.value = String.fromCodePoint(val as number).toUpperCase()
				break
		}
		// 'image' type simply falls through, as those don't need to be set
	}

	// Propagate options panel changes through
	// options-panel -> options -> OptionModel
	applyChange(event: Event) {
		const el = event.target as HTMLInputElement,
			id = el.getAttribute('id'),
			model = models[id]

		// Not an option input element
		if (!model) {
			return
		}

		let val: boolean | string | number
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
				val = el.value.toUpperCase().codePointAt(0)
				break
			case optionType.image:
				// Not recorded. Extracted directly by the background handler
				loadModule('background').then(m =>
					m.store((event as any).target.files[0]))
				return
		}

		if (!model.validate(val)) {
			el.value = ''
		} else {
			options[id] = val
		}
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
		this.import.click()
		const handler = () =>
			this.importConfigFile()
		this.import.addEventListener("change", handler, { once: true })
	}

	// After the file has been uploaded, parse it and import the configs
	async importConfigFile() {
		const reader = new FileReader()
		reader.readAsText(this.import.files[0])
		const event = await load(reader) as any

		// In case of corruption
		let json: { [key: string]: string }
		try {
			json = JSON.parse(event.target.result)
		}
		catch (err) {
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

	// Render Hidden posts counter
	renderHidden(count: number) {
		if (!this.isRendered) {
			return
		}
		write(() => {
			const el = this.hidden
			el.textContent = el.textContent.replace(/\d+$/, count.toString())
		})
	}
}
