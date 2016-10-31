// Miscellaneous post component rendering functions

import {page, mine} from '../../state'
import {posts as lang} from '../../lang'
import {PostLinks} from '../models'

// Render a link to other posts
export function renderPostLink(num: number, board: string, op: number): string {
	let text = num.toString(),
		url = "#p" + text

	if (op !== page.thread) {         // Cross-thread
		text += " \u27a1"
		url = op + url
	} else if (num === page.thread) { // OP of this thread
		text += " " + lang.OP
	}
	if (board !== page.board) {       // Cross-board
		text = `>/${board}/` + text
		url = `/${board}/` + url
	}
	if (mine.has(num)) {              // Post, I made
		text += ' ' + lang.you
	}

	return `<a class="history" href="${url}">>>${text}</a>`
}

// TODO: Reimplement, when moderation done

// Render USER WAS BANNED FOR THIS POST message, or similar
// export function renderBanned(): string {
// 	return `<b class="admin banMessage">${lang.mod.banMessage}</b>`
// }
//
// Render moderation information. Only exposed to authenticated staff.
// export function renderModInfo(info) {
// 	let html = '<b class="modLog admin">'
// 	for (let action of info) {
// 		html += `${lang.mod.formatLog(action)}<br>`
// 	}
// 	html += '</b>'
// 	return html
// }

// Render links to posts that are linking to the target post
export function renderBacklinks(links: PostLinks): string {
	if (!links) {
		return ''
	}
	let html = ''
	for (let id in links) {
		const {board, op} = links[id]
		if (html) {
			html += ' '
		}
		html += renderPostLink(parseInt(id), board, op)
	}
	return html
}
