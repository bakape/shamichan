// Highlight options button by fading out and in, if no options are set
(function() {
	if (localStorage.optionsSeen) {
		return
	}
	const el = document.query('#options')
	el.style.opacity = 1
	let out = true,
		clicked
	el.addEventListener("click", () => {
		clicked = true
		localStorage.optionsSeen = 1
	})
	tick()

	function tick() {
		// Stop
		if (clicked) {
		    el.style.opacity = 1
			return
		}

    	el.style.opacity = +el.style.opacity + (out ? -0.02 : 0.02)
		const now = +el.style.opacity

		// Reverse direction
		if ((out && now <= 0) || (!out && now >= 1)) {
		    out = !out
		}
		requestAnimationFrame(tick)
	}
})()

// View of the options panel
export default ModalView.extend({
	id: 'options-panel',

	events: {
		'click .option_tab_sel>li>a': 'switchTab',
		'change': 'applyChange',
		'click #export': 'export',
		'click #import': 'import',
		'click #hidden': 'clearHidden'
	},

	/**
	 * Render the options panel. Only called once on page load.
	 */
	render() {
	    this.el.innerHTML = render()
		this.assignValues()
		this.hidden = this.el.query('#hidden')
		events.reply('hide:render', this.renderHidden, this)
	},

	/**
	 * Assign loaded option settings to the respective elements in the options
	 * panel
	 */
	assignValues() {
		for (let id in optionModels) {
			const model = optionModels[id],
				el = this.el.query('#' + id)
			const {type} = model,
				val = model.get()
			if (type === 'checkbox') {
			    el.checked = val
			} else if (type === 'number' || type instanceof Array) {
			    el.value = val
			} else if (type === 'shortcut') {
			    el.value = String.fromCharCode(val).toUpperCase()
			}

			// 'image' type simply falls through, as those don't need to be set
		}
	},

	/**
	 * Switch to a tab, when clicking the tab butt
	 * @param {Event} event
	 */
	switchTab(event) {
		event.preventDefault()
		const el = event.target

		// Deselect previous tab
		each(this.el.children, el =>
			el.query('.tab_sel').classList.remove('tab_sel'))

		// Select the new one
		el.classList.add('tab_sel')
		find(this.el.lastChild.children, li =>
			li.classList.contains(el.getAttribute('data-content'))
		)
			.classList.add('tab_sel')
	},

	/**
	 * Propagate options panel changes through
	 * options -> optionModels -> localStorage
	 * @param {Event} event
	 */
	applyChange(event) {
		const el = event.target,
			id = el.getAttribute('id'),
			model = optionModels[id]
		let val
		switch (model.type) {
			case 'checkbox':
				val = el.checked
				break
			case 'number':
				val = parseInt(el.value)
				break
			case 'image':
				// Not recorded. Extracted directly by the background handler.
				return events.request('background:store', event.target)
			case 'shortcut':
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
	},

	/**
	 * Dump options to JSON file and upload to user
	 */
	export() {
		const a = document.getElementById('export')
		a.setAttribute('href', window.URL
			.createObjectURL(new Blob([JSON.stringify(localStorage)], {
				type: 'octet/stream'
			}))
		)
		a.setAttribute('download', 'meguca-config.json')
	},

	/**
	 * Import options from uploaded JSON file
	 * @param {Event} event
	 */
	import(event) {
		// Proxy to hidden file input
		event.preventDefault()
		const el = document.query('#importSettings')
		el.click()
		util.once(el, 'change', () => {
			var reader = new FileReader()
			reader.readAsText(input.files[0])
			reader.onload = event => {
				// In case of curruption
				let json
				try {
					json = JSON.parse(event.target.result)
				}
				catch(err) {
					alert('Import failed. File corrupt')
					return
				}
				localStorage.clear()
				for (let key in json) {
					localStorage[key] = json[key]
				}
				alert('Import successfull. The page will now reload.')
				location.reload()
			};
		})
	},

	/**
	 * Render Hiden posts counter
	 * @param {int} count
	 */
	renderHidden(count) {
		const el = this.hidden
		el.textContent = el.textContent.replace(/\d+$/, count)
	},

	/**
	 * Clear displayed hidden post counter
	 */
	clearHidden() {
		main.request('hide:clear')
		this.renderHidden(0)
	}
})
