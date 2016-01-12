import {filter} from '../../vendor/underscore'
import {parseHTML, parseAttributes} from '../util'
import options from "./opts"
import ln from 'lang'

const lang = ln.opts

/**
 * Render the inner HTML of the options panel
 */
export default function () {
	let html = '<ul class="option_tab_sel">'
	const {tabs} = lang,
		opts = []

	// Render tab butts
	for (let i = 0; i < tabs.length; i++) {
		// Pick the options for this specific tab, according to current
		// template and server configuration
		opts[i] = filter(options, opt =>
			opt.tab === i
				&& (opt.load === undefined  || opt.load)
				&& !opt.hidden)

		if (!opts[i].length) {
			continue
		}
		const attrs = {
			'data-content': `tab-${i}`,
			class: 'tab_link'
		}

		// Highlight the first tabButt by default
		if (i === 0) {
			attrs.class += ' tab_sel'
		}
		html += parseHTML
			`<li>
				<a ${parseAttributes(attrs)}>
					${tabs[i]}
				</a>
			</li>`
	}

	html += '</ul><ul class="option_tab_cont">'
	for (let i = 0; i < opts.length; i++) {
		html += renderTab(opts[i], i)
	}
	html += '</ul>'

	return html
}

/**
 * Render tab contents
 */
function renderTab(opts, i) {
	if (!opts.length) {
		return ''
	}
	let html = ""
	html += `<li class="tab-${i}`

	// Show the first tab by default
	if (i === 0) {
		html += ' tab_sel'
	}
	html += '">'

	// Render the actual options
	for (let opt of opts) {
		html += renderOption(opt)
	}

	if (i === 0) {
		html += renderExtras()
	}
	html += '</li>'

	return html
}

/**
 * Render a single option from it's schema
 */
function renderOption(opt) {
	let html = ''
	const isShortcut = opt.type === 'shortcut',
		isList = opt.type instanceof Array,
		isCheckbox = opt.type === 'checkbox' || opt.type === undefined,
		isNumber = opt.type === 'number',
		isImage = opt.type === 'image'
	if (isShortcut) {
		html += 'Alt+'
	}
	if (!isList) {
		html += '<input'
		if (isCheckbox || isImage)
			html += ` type="${(isCheckbox ? 'checkbox' : 'file')}"`
		else if (isNumber)
			html += ' style="width: 4em;" maxlength="4"'
		else if (isShortcut)
			html += ' maxlength="1"'
	} else {
		html += '<select'
	}

	const [label,title] = lang.labels[opt.id]
	html += ` id="${opt.id}" title="${title}">`

	if (isList) {
		for (let item of opt.type) {
			html += parseHTML
				`<option value="${item}">
					${lang.modes[item] || item}
				</option>`
		}
		html += '</select>'
	}
	html += parseHTML
		`<label for="${opt.id}" title="${title}">
			${label}
		</label>
		<br>`

	return html
}

/**
 * Hidden post reset, Export and Import links to first tab
 */
function renderExtras() {
	let html = '<br>'
	const links = ['export', 'import', 'hidden']
	for (let id of links) {
		const [label, title] = lang.labels[id]
		html += parseHTML
			`<a id="${id}" title="${title}">
				${label}
			</a> `
	}

	// Hidden file input for uploading the JSON
	const attrs = {
		type: 'file',
		id: 'importSettings',
		name: "Import Settings"
	}
	html += `<input ${parseAttributes(attrs)}>`

	return html
}
