// Miscalleneous post component rendering functions

import {page, mine} from '../../state'
import {posts as lang} from '../../lang'
import {PostLinks} from '../models'

// Render a link to other posts
export function renderPostLink(num: number, board: string, op: number): string {
	let text = '>>'
	if (board !== page.board) { // Cross-board
		text += `>/${board}/`
	}
	text += num
	if (mine.has(num)) { // Post, I made
		text += ' ' + lang.you
	}
	const {thread} = page
	if (op !== thread) { // Cross-thread
		text += ' \u27a1'
	} else if (num == thread) { // OP post of this thread
		text += ' ' + lang.OP
	}
	return postAnchor(`/${board}/${op}#${num}`, text)
}

// Render the anchor element of a post link
function postAnchor(href: string, text: string): string {
	return `<a class="history" href="${href}">${text}</a>`
}

// TODO: Reimplement, when moderation done

// Render USER WAS BANNED FOR THIS POST message, or similar
// export function renerBanned(): string {
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
