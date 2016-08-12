import {config} from '../../state'
import {renderPostLink} from './etc'
import {PostData, PostLinks, TextState} from '../models'
import {escape} from '../../util'

// Map of {name: url} for generating `>>>/foo/bar` type reference links
let refTargets: StringMap

// Render the text body of a post
export function renderBody(data: PostData): string {
	if (data.editing) {
		return parseOpenBody(data)
	}
	return parseClosedBody(data)
}

// Parse a text body of a closed post
function parseClosedBody(data: PostData): string {
	data.state = {
		spoiler: false,
		quote: false,
	}
	let html = ""
	for (let line of data.body.split("\n")) {
		html += parseTerminatedLine(line, data)
	}
	if (data.state.spoiler) { // Close spoiler tag on post end
		html += '</del>'
	}
	data.state = null // Clean up a bit

	return html
}

// Parse a text body, that is still being editted
function parseOpenBody(data: PostData): string {
	const state: TextState = data.state = {
		spoiler: false,
		quote: false,
	}
	let html = ""
	const lines = data.body.split("\n")
	for (let i = 0; i < lines.length - 1; i++) {
		html += parseTerminatedLine(lines[i], data)
	}

	state.line = lines[lines.length - 1]
	html += parseOpenLine(state)
	if (state.spoiler) {
		html += '</del>'
	}

	return html
}

// Parse a single terminated line
function parseTerminatedLine(line: string, data: PostData): string {
	let html = ""
	if (line[0] === ">") {
		data.state.quote = true
		html += "<em>"
	} else if (line[0] === "#") {

		// TODO: Hash command rendering

		const m = line.match(/^#(flip|\d*d\d+|8ball)$/)
		if (m) {
			return line
		}
	}

	// Check for spoilers
	while (true) {
		const i = line.indexOf("**")
		html += parseFragment(line.substring(i), data)
		if (i !== -1) {
			html += `<${data.state.spoiler ? '/' : ''}del>`
			data.state.spoiler = !data.state.spoiler
			line = line.substring(i + 1)
		} else {
			break
		}
	}

	if (data.state.quote) {
		data.state.quote = false
		html += "</em>"
	}
	return html + "<br>"
}

// Parse a line that is still being editted
function parseOpenLine(state: TextState): string {
	let html = ""
	if (state.line[0] === ">") {
		state.quote = true
		html += "<em>"
	}

	// Check for spoilers
	let {line} = state
	while (true) {
		const i = line.indexOf("**")
		html += line.substring(i)
		if (i !== -1) {
			html += `<${state.spoiler ? '/' : ''}del>`
			state.spoiler = !state.spoiler
			line = line.substring(i + 1)
		} else {
			break
		}
	}

	// Close quote in progress
	if (state.quote) {
		html += '</em>'
	}

	return html
}

// Parse a line fragment
function parseFragment(frag: string, data: PostData): string {
	let html = ""
	for (let word of frag.split(" ")) {
		if (!frag) {
			continue
		}
		if (word[0] === ">") {
			if (/^>>\d+$/.test(word)) {
				// Post links
				html += parsePostLink(word, data.links)
				continue
			} else if (/^>>>\/\w+\//.test(word)) {
				// Internal and custom reference URLs
				html += parseReference(word)
				continue
			}
		} else if (word.startsWith("http")) {
			// Generic URLs
			html += parseURL(word)
			continue
		}
		html += escape(word)
	}

	return html
}

// Verify and render a link to other posts
function parsePostLink(bit: string, links: PostLinks): string {
	if (!links) {
		return bit
	}
	const num = parseInt(bit.match(/^>>\/(\d+)$/)[1]),
		verified = links[num]
	if (!verified) {
		return bit
	}
	return renderPostLink(num, verified.board, verified.op)
}

// Generate all possible refference name and link pairs for externa
// `>>>/foo/bar` links
export function genRefTargets() {
	const targets: StringMap = {}

	for (let name in config.links) {
		targets[name] = config.links[name]
	}
	for (let board of config.boards) { // Boards override links
		targets[board] = `../${board}/`
	}

	refTargets = targets
}

genRefTargets()

// Parse internal or customly set reference URL
function parseReference(bit: string): string {
	const name = bit.match(/^>>>\/(\w+)\/$/)[1],
		href = refTargets[name]
	if (!href) {
		return escape(bit)
	}
	return newTabLink(href, bit)
}

// Render and anchor link that opens in a new tab
function newTabLink(href: string, text: string): string {
	return `<a href="${encodeURI(href)}" target="_blank">${escape(text)}</a>`
}

// Render generic URLs and embed, if aplicable
function parseURL(bit: string): string {

	// TODO: Embeds

	if (/^https?:\/\/[^-A-Za-z0-9+&@#/%?=~_:\.]$/.test(bit)) {
		return newTabLink(bit, bit)
	}

	return escape(bit)
}
