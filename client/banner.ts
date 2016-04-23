// Handles all things related to the top banner

import {config} from './state'
import {defer} from './defer'

// Reders the HTML of the FAQ panel
export function renderFAQ() {
	let html = '<ul>'
	for (let line of config.FAQ) {
		html += `<li>${line}</line>`
	}
	html += `</ul>`
	document.query('#FAQ-panel').innerHTML = html
}

defer(renderFAQ)
