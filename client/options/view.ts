import {BannerModal} from '../modal'
import renderContents from './render'
import {models, default as options} from '../options'
import {optionType} from './specs'
import {each, find} from 'underscore'
import {onceAll} from '../util'
import {opts as lang} from '../lang'

// View of the options panel
export default class OptionsPanel extends BannerModal {
	$hidden: Element

	constructor() {
		super({id: 'options-panel'})
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
		this.el.innerHTML = renderContents()
		this.assignValues()
		this.$hidden = this.el.query('#hidden')

		// TODO: Hidden posts count rendering
		// events.reply('hide:render', this.renderHidden, this)
	}

	// Assign loaded option settings to the respective elements in the options
	// panel
	assignValues() {
		for (let id in models) {
			const model = models[id],
				el = this.el.query('#' + id),
				{type} = model.spec,
				val = model.get()
			if (type === optionType.checkbox) {
				el.checked = val as boolean
			} else if (type === optionType.number || type === optionType.menu) {
				el.value = val
			} else if (type === optionType.shortcut) {
				el.value = String.fromCharCode(val as number).toUpperCase()
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

			/*
			TODO: System.import().then()
			case 'image':
				// Not recorded. Extracted directly by the background handler.
				return events.request('background:store', event.target)
			*/

			case optionType.shortcut:
				val = el.value.toUpperCase().charCodeAt(0)
				break
			default:
				val = el.value
		}

		if (!model.validate(val)) {
			el.value = ''
		} else {
			options.set(id, val)
		}
	}

	// Switch to a tab, when clicking the tab butt
	switchTab(event: Event) {
		event.preventDefault()
		const el = event.target as Element

		// Deselect previous tab
		each(this.el.children, el =>
			el.query('.tab_sel').classList.remove('tab_sel'))

		// Select the new one
		el.classList.add('tab_sel')
		find(this.el.lastChild.children, li =>
			li.classList.contains(el.getAttribute('data-content'))
		)
			.classList.add('tab_sel')
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
		event.preventDefault()
		const el = document.query('#importSettings')
		el.click()
		onceAll(el, 'change', () => {
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
		const el = this.$hidden
		el.textContent = el.textContent.replace(/\d+$/, count.toString())
	}

	// Clear displayed hidden post counter
	clearHidden() {

		// TODO: Fix  after psot hiding ported
		// main.request('hide:clear')

		this.renderHidden(0)
	}
}

// Highlight options button by fading out and in, if no options are set
(function() {
	if (localStorage.getItem('optionsSeen')) {
		return
	}
	const el = document.query('#options')
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
})()
