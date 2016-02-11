import {config} from '../state'
import {escape} from 'underscore'
import {renderPostLink} from './etc'

// Render the text body of a post
export function renderBody(data) {
	if (!data.state) {
		// Initial post state [new_line, no_qoute, no_spoiler]
		data.state = [0, 0, 0]
	}
	let html = renderFragment(data.body, data.state, data.dice)
	if (data.state[1]) { // Close quote on post end
		html += '</em>'
	}
	if (data.state[2]) { // Same with spoilers
		html += '</del>'
	}
	return html
}

// Parse commited text body fragment
export function renderFragment(frag, data) {
	const lines = frag.split('\n'),
		{state} = data
	let html = ''
	for (let i = 0; i < lines.length; i++) {
		// Start a new line
		if (state[0] && i % 2) {
			// Close quoute
			if (state[1] % 2) {
				html += '</em>'
				state[1]++
			}
			html += '<br>'
			state[0] = 0
		}

		// Quote or line starts with link/embed
		const line = lines[i]
		if (!state[0] && line.startsWith('>')) {
			html += '<em>'
			state[1]++
		}

		// Bodies may be empty
		if (frag) {
			for (let word of line.split(' ')) {
				html += parseWord(word, data)
				state[0] = 1
			}
		}
	}
	return html
}

// Convert a word to it's appropriate HTML representation
function parseWord(word, data) {
	// `[spoiler]` and `[/spoiler]` are treated the same way. You can't nest
	// them.
	const split = word.split(/\[\/?spoiler]/i)
	let html = ''
	for (let i = 0; i < split.length; i++) {
		// Insert spoiler tags
		if (i % 2) {
			html += `<${data.state[2]++ % 2 ? '/' : ''}del>`

			// TODO: Do we need special logic for postForms here?
		}

		const bit = split[i]
		if (/^>>\d+$/.test(bit)) {
			// Post links
			html += parsePostLink(bit, data.links)
		} else if (/^>>>\/\w+\//.test(bit)) {
			// Internal and custom reference URLs
			html += parseReference(bit)
		} else if (/^https?:\/\/[^-A-Za-z0-9+&@#/%?=~_]$/.test(bit)) {
			// Generic URLs
			html += parseURL(bit)
		} else if (/<strong>.+<\/strong>/.test(bit)) {
			// Hash command results. Already verified server-side.
			html += bit
		} else {
			html += escape(bit)
		}
	}
	return html
}

// Verify and render a link to other posts
function parsePostLink(bit, links) {
	if (!links) {
		return bit
	}
	const num = bit.match(/^>>\/(\d+)$/)[1],
		verified = links[num]
	if (!verified) {
		return bit
	}
	return renderPostLink(num, verified.board, verified.OP)
}

// Generate all possible refference name and link pairs
const refTargets = {},
	{boards} = config
for (let board of boards.enabled) {
	refTargets[board] = `../${board}/`
}
for (let [name, link] of boards.psuedo.concat(boards.links)) {
	refTargets[name] = link
}

// Parse internal or customly set reference URL
function parseReference(bit) {
	const name = bit.match(/^>>>\/(\w+)\/$/)[1],
		href = refTargets[name]
	if (!href) {
		return bit
	}
	return newTabLink(href, bit)
}

// Render and anchor link that opens in a new tab
function newTabLink(href, text) {
	return `<a href="${href}" target="_blank">${text}</a>`
}

// Render generic URLs and embed, if aplicable
function parseURL(bit) {

	// TODO: Embeds

	return newTabLink(encodeURI(href), bit)
}
