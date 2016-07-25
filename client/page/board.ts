import {HTML, random, escape} from '../util'
import {posts as lang} from '../lang'
import {boardConfig, page} from '../state'
import {write} from '../render'
import {$threads} from './common'

// Button for expanding the thread creation form
const newThreadButton = HTML
	`<aside class="act glass posting">
		<a class="new-thread-button">
			${lang.newThread}
		</a>
	</aside>`

// Format a board name and title into cannonical board header format
export const formatHeader = (name: string, title: string): string =>
	escape(`/${name}/ - ${title}`)

export function renderBoard() {
	let html = ""
	if (boardConfig.banners.length) {
		const banner = random(boardConfig.banners)
		html += `<h1><img src="/assets/banners/${banner}"></h1>`
	}
	html += HTML
		`<h1>
			${formatHeader(page.board, boardConfig.title)}
		</h1>
		${newThreadButton}
		<hr>
		// TODO: Catalog
		<hr>
		${newThreadButton}`
	write(() =>
		$threads.innerHTML = html)
}
