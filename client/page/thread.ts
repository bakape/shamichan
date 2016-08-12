import {HTML, escape} from '../util'
import {navigation as lang} from '../lang'
import {ThreadData} from '../posts/models'
import {page} from '../state'
import {write, $threads} from '../render'
import renderPost from '../posts/render/posts'

// Render the HTML of a thread page
export default function renderThread(thread: ThreadData) {

	// TODO: Apply thread title as tab title

	const title = `/${page.board}/ - ${escape(thread.subject)} (#${thread.id})`

	const html = HTML
		`<h1>
			${title}
		</h1>
		<span class="act">
			<a href="#bottom">
				${lang.bottom}
			</a>
		</span>
		<span class="act">
			<a id="expand-images">
				${lang.expand}
			</a>
		</span>
		<hr>
		<div id="thread-container"></div>
		<hr>
		<span class="act">
			<a href="." class="history">
				${lang.return}
			</a>
		</span>
		<span class="act">
			<a href="#">
				${lang.top}
			</a>
		</span>
		<span id="lock">
			${lang.lockedToBottom}
		</span>`
}

function renderPosts(thread: ThreadData): string {
	return renderPost(thread)
}
