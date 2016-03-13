/*
 Renders the HTML of the options panel
*/

import {filter, extend} from 'underscore'
import {parseHTML, parseAttributes, ElementAttributes} from '../util'
import {opts as lang, OptLabel} from '../lang'
import {specs, OptionSpec, optionType} from './specs'
import {OptionID} from '../options'

// Render the inner HTML of the options panel
export default function (): string {
	let html = '<ul class="option_tab_sel">'
	const {tabs} = lang,
		opts: OptionSpec[][] = []

	// Render tab butts
	for (let i = 0; i < tabs.length; i++) {
		// Pick the options for this specific tab, according to current
		// template and server configuration
		opts[i] = filter<OptionSpec>(specs, spec =>
			spec.tab === i && !spec.noLoad && !spec.hidden)

		if (!opts[i].length) {
			continue
		}
		const attrs: ElementAttributes = {
			'data-content': `tab-${i}`,
			class: 'tab_link'
		}

		// Highlight the first tabButt by default
		if (i === 0) {
			attrs['class'] += ' tab_sel'
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

// Render tab contents
function renderTab(opts: OptionSpec[], i: number): string {
	if (!opts.length) {
		return ''
	}
	let html = ''
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

// Render a single option from it's schema
function renderOption(spec: OptionSpec): string {
	switch (spec.type) {
		case optionType.shortcut:
			return 'Alt+' + renderInput(spec.id, {maxlength: '1'})
		case optionType.checkbox:
			return renderInput(spec.id, {type: 'checkbox'})
		case optionType.number:
			return renderInput(spec.id, {
				style: 'width: 4em;',
				maxlength: '4'
			})
		case optionType.image:
			return renderInput(spec.id, {type: 'file'})
		case optionType.menu:
			return renderMenu(spec)
	}
}

// Common input field render logic
function renderInput(id: OptionID, attrs: ElementAttributes): string {
	const [label,title] = lang.labels[id]
	extend(attrs, {id, title})
	return `<input ${parseAttributes(attrs)}>` + renderLabel(id, title, label)
}

// Render the description label to the right of the option
function renderLabel(id: OptionID, title: string, label: string): string {
	return parseHTML
		`<label for="${id}" title="${title}">
			${label}
		</label>
		<br>`
}

// Render drop down selection menu
function renderMenu({id, list}: OptionSpec): string {
	const [label, title] = lang.labels[id]
	let html = '<select id="${id}" title="${title}">'
	for (let item of list) {
		html += parseHTML
			`<option value="${item}">
				${lang.modes[item] || item}
			</option>`
	}
	html += '</select>' + renderLabel(id, title, label)
	return html
}

// Hidden post reset, Export and Import links to first tab
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
	const attrs: ElementAttributes = {
		type: 'file',
		id: 'importSettings',
		name: "Import Settings"
	}
	html += `<input ${parseAttributes(attrs)}>`

	return html
}
