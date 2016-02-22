import {parseHTML} from '../../util'
import {renderHeader} from './header'
import {renderImage} from './image'
import {renderBanned, renderBacklinks} from './etc'
import {renderBody} from './body'

// Render the OP
export function renderSection(data, cls = '') {
	if (data.locked) {
		cls += ' locked'
	}
	if (data.editing) {
		cls += ' editing'
	}
	data.largeThumb = true // Larger thumbnails

	return parseHTML
		`<section id="p${data.num}" class="${cls}">
			<div class="background glass">
				${renderPost(data)}
				<span class="omit"></span>
			</div>
		</section>`
}

// Render a reply post
export function renderArticle(data) {
	let cls = 'glass'
	if (data.editing) {
		cls += ' editing'
	}
	return parseHTML
		`<article id="p${data.num}" class="${cls}">
			${renderPost(data)}
		</article>`
}

function renderPost(data) {
	const {mod, body, backlinks, banned} = data

	return parseHTML
		`${renderHeader(data)}
		${renderImage(data)}
		<div class="container">
			${mod ? renderModInfo(mod) : ''}
			<blockquote>
				${renderBody(body)}
			</blockquote>
			<small>
				${renderBacklinks(backlinks)}
			</small>
			${banned ? renderBanned(): ''}
		</div>`
}
