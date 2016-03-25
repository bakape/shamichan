import {parseHTML} from '../../util'
import {renderHeader} from './header'
import {renderImage} from './image'
import {renderBacklinks} from './etc'
import {renderBody} from './body'
import {PostData} from '../models'

// Render the OP
export function renderSection(data: PostData, cls: string = ''): string {
	if (data.locked) {
		cls += ' locked'
	}
	if (data.editing) {
		cls += ' editing'
	}
	data.largeThumb = true // Larger thumbnails

	return parseHTML
		`<section id="p${data.id.toString()}" class="${cls}">
			<div class="background glass">
				${renderPost(data)}
				<span class="omit"></span>
			</div>
		</section>`
}

// Render a reply post
export function renderArticle(data: PostData): string {
	let cls = 'glass'
	if (data.editing) {
		cls += ' editing'
	}
	return parseHTML
		`<article id="p${data.id.toString()}" class="${cls}">
			${renderPost(data)}
		</article>`
}

function renderPost(data: PostData): string {
	const {body, backlinks} = data

	return parseHTML
		`${renderHeader(data)}
		${data.image ? renderImage(data.image): ''}
		<div class="container">
			<blockquote>
				${renderBody(data)}
			</blockquote>
			<small>
				${renderBacklinks(backlinks)}
			</small>
		</div>`
}
