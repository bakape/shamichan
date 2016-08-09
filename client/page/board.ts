import {HTML, random, escape} from '../util'
import {posts as lang, navigation} from '../lang'
import {BoardConfigs} from '../state'
import {ThreadData} from '../posts/models'
import {renderThumbnail} from '../posts/render/image'
import options from '../options'

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

// Render a board page's HTML
export default function renderBoard(
	board: string,
	conf: BoardConfigs,
	threads: ThreadData[]
): string {
	let html = ""
	if (conf.banners.length) {
		const banner = random(conf.banners)
		html += `<h1><img src="/assets/banners/${banner}"></h1>`
	}
	html += HTML
		`<h1>
			${formatHeader(board, conf.title)}
		</h1>
		${newThreadButton}
		<hr>
		${renderCatalog(threads)}
		<hr>
		${newThreadButton}`

	return html
}

// Render the thread catalog
function renderCatalog(threads: ThreadData[]): string {
	let html = `<div id="catalog">`
	for (let thread of threads) {
		html += renderThread(thread)
	}
	html += "</div>"

	return html
}

// Render a single thread for the thread catalog
function renderThread(thread: ThreadData): string {
	const href = `../${thread.board}/${thread.id}`,
		lastN = options.lastN.toString()

	return HTML
	`<article class="glass">
		${thread.image ? renderThumbnail(thread.image, href)  + "<br>" : ""}
		<small class="thread-links">
			<span title="${navigation.catalogOmit}">
				${thread.postCtr.toString()}/${thread.imageCtr.toString()}
			</span>
			<span class="act">
				<a href="${href}" class="history">
					${navigation.expand}
				</a>
				] [
				<a href="${href}?last=${lastN}" class="history">
					${navigation.last} ${lastN}
				</a>
			</span>
		</small>
		<br>
		<h3>
			「${escape(thread.subject)}」
		</h3>
	</article>`
}
