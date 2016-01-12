import {page, mine} from '../../state'
import lang from 'lang'

/**
 * Render a link to other posts
 * @param {(int|string)} num
 * @param {string} board
 * @param {int} op
 * @returns {string}
 */
export function renderPostLink(num, board, op) {
	let text = '>>'
	if (board !== page.get('board')) { // Cross-board
		text += `>/${board}/`
	}
	text += num
	if (mine.get(num)) { // Post, I made
		text += ' ' + lang.you
	}
	const thread = page.get('thread')
	if (op !== thread) { // Cross-thread
		text += ' \u27a1'
	} else if (num == thread) { // OP post of this thread
		text += ' ' + lang.OP
	}
	return postAnchor(`../${board}/${op}#${num}`, text)
}

/**
 * Render the anchor element of a post link
 * @param {string} href
 * @param {string} href
 * @returns {string}
 */
function postAnchor(href, text) {
	return parseHTML
		`<a class="history" href="${href}">
			${text}
		</a>`
}

/**
 * Render USER WAS BANNED FOR THIS POST message, or similar
 * @returns {string}
 */
export function renerBanned() {
	return `<b class="admin banMessage">${lang.mod.banMessage}</b>`
}

/**
 * Render moderation information. Only exposed to authenticated staff.
 * @param {Object[]} info
 * @returns {string}
 */
export function renderModInfo(info) {
	let html = '<b class="modLog admin">'
	for (let action of info) {
		html += `${lang.mod.formatLog(action)}<br>`
	}
	html += '</b>'
	return html
}

/**
 * Render links to posts that are linking to the target post
 * @returns {Object} links
 * @returns {string}
 */
export function renderBacklinks(links) {
	if (!links) {
		return ''
	}
	let html = ''
	for (let {board, op} in links) {
		if (html) {
			html += ' '
		}
		html += renderPostLink(links[num], board, op)
	}
	return html
}
