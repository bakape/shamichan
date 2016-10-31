import { config, boards, boardConfig } from '../../state'
import { renderPostLink } from './etc'
import { PostData, PostLinks, TextState } from '../models'
import { escape } from '../../util'
import { deferInit } from "../../defer"
import { parseEmbeds } from "../embed"

// Map of {name: url} for generating `>>>/foo/bar` type reference links
let refTargets: { [key: string]: string }

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
		iDice: 0,
	}
	let html = ""
	for (let line of data.body.split("\n")) {
		html += parseTerminatedLine(line, data)
	}
	data.state = null // Clean up a bit

	return html
}

// Parse a text body, that is still being edited
function parseOpenBody(data: PostData): string {
	const state: TextState = data.state = {
		spoiler: false,
		quote: false,
		iDice: 0,
	}
	let html = ""
	const lines = data.body.split("\n")
	for (let i = 0; i < lines.length - 1; i++) {
		html += parseTerminatedLine(lines[i], data)
	}

	state.line = lines[lines.length - 1]
	html += parseOpenLine(state)

	return html
}

// Parse a single terminated line
export function parseTerminatedLine(line: string, data: PostData): string {
	// For hiding redundant newlines using CSS
	if (line === "") {
		return "<br>"
	}

	let html = "<span>"
	const {state} = data
	state.spoiler = state.quote = false
	if (line[0] === ">") {
		state.quote = true
		html += "<em>"
	} else if (line[0] === "#") {
		const m = line.match(/^#(flip|\d*d\d+|8ball|pyu|pcount)$/)
		if (m) {
			return html + parseCommand(m[1], data) + terminateTags(state, true)
		}
	}

	// Check for spoilers
	if (boardConfig.spoilers) {
		while (true) {
			const i = line.indexOf("**")
			if (i !== -1) {
				html += parseFragment(line.slice(0, i), data)
					+ `<${state.spoiler ? '/' : ''}del>`
				state.spoiler = !state.spoiler
				line = line.substring(i + 2)
			} else {
				html += parseFragment(line, data)
				break
			}
		}
	} else {
		html += parseFragment(line, data)
	}

	html += terminateTags(state, true)

	return html
}

// Close all open tags at line end
function terminateTags(state: TextState, newLine: boolean): string {
	let html = ""
	if (state.spoiler) {
		html += "</del>"
	}
	if (state.quote) {
		html += "</em>"
	}
	if (newLine) {
		html += "<br>"
	}
	return html + "</span>"
}

// Parse a line that is still being edited
export function parseOpenLine(state: TextState): string {
	let html = "<span>"
	if (state.line[0] === ">") {
		state.quote = true
		html += "<em>"
	}

	// Check for spoilers
	if (boardConfig.spoilers) {
		let {line} = state
		while (true) {
			const i = line.indexOf("**")
			if (i !== -1) {
				html += escape(line.slice(0, i))
					+ `<${state.spoiler ? '/' : ''}del>`
				state.spoiler = !state.spoiler
				line = line.slice(i + 2)
			} else {
				html += escape(line.substring(i))
				break
			}
		}
	} else {
		html += escape(state.line)
	}

	html += terminateTags(state, false)

	return html
}

// Parse a line fragment
function parseFragment(frag: string, data: PostData): string {
	let html = ""
	for (let word of frag.split(" ")) {
		if (html) {
			html += " "
		}
		if (!word) {
			html += " "
			continue
		}
		if (word[0] === ">") {
			if (/^>{2,}\d+$/.test(word)) {
				// Post links
				html += parsePostLink(word, data.links)
				continue
			} else if (/^>{3,}\/\w+\/$/.test(word)) {
				// Internal and custom reference URLs
				html += parseReference(word)
				continue
			}
		} else if (word.startsWith("http") || word.startsWith("magnet:?")) {
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
		return escape(bit)
	}
	const [, extraQuotes, id] = bit.match(/^>>(>*)(\d+)$/),
		num = parseInt(id),
		verified = links[num]
	if (!verified) {
		return escape(bit)
	}
	return escape(extraQuotes)
		+ renderPostLink(num, verified.board, verified.op)
}

// Generate all possible reference name and link pairs for external
// `>>>/foo/bar` links
export function genRefTargets() {
	const targets: { [key: string]: string } = {}

	for (let name in config.links) {
		targets[name] = config.links[name]
	}
	for (let board of boards) { // Boards override links
		targets[board] = `../${board}/`
	}

	refTargets = targets
}

// Parse internal or customly set reference URL
function parseReference(bit: string): string {
	const [, extraQuotes, name] = bit.match(/^>>>(>*)\/(\w+)\/$/),
		href = refTargets[name]
	if (!href) {
		return escape(bit)
	}
	return escape(extraQuotes) + newTabLink(href, bit)
}

// Render and anchor link that opens in a new tab
function newTabLink(href: string, text: string): string {
	return `<a href="${encodeURI(href)}" target="_blank">${escape(text)}</a>`
}

// Render generic URLs and embed, if applicable
function parseURL(bit: string): string {
	const embed = parseEmbeds(bit)
	if (embed) {
		return embed
	}

	const m = bit
		.match(/^(magnet:\?|https?:\/\/)[-a-zA-Z0-9@:%_\+\.~#\?&\/=]+$/)
	if (m) {
		if (m[1].startsWith("magnet")) {
			return escape(bit).link(encodeURI(bit))
		}
		return newTabLink(bit, bit)
	}

	return escape(bit)
}

// Parse a hash command
function parseCommand(bit: string, {commands, state}: PostData): string {
	// Guard against the first command being an invalid dice roll and parsing
	// lines in the post form.
	if (!commands || !commands[state.iDice]) {
		return "#" + bit
	}

	// TODO: Sycnwatch

	let inner: string
	switch (bit) {
		case "flip":
		case "8ball":
		case "pyu":
		case "pcount":
			inner = commands[state.iDice++].val.toString()
			break
		default:
			// Validate dice
			const m = bit.match(/(\d*)d(\d+)/)
			if (parseInt(m[1]) > 10 || parseInt(m[2]) > 100) {
				break
			}

			const rolls = commands[state.iDice++].val as number[]
			inner = ""
			let sum = 0
			for (let i = 0; i < rolls.length; i++) {
				if (i) {
					inner += " + "
				}
				sum += rolls[i]
				inner += rolls[i]
			}
			if (rolls.length > 1) {
				inner += " = " + sum
			}
	}

	if (inner) {
		return `<strong>#${bit} (${inner})</strong>`
	}
	return "#" + bit
}

deferInit(genRefTargets)
