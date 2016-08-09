import {HTML} from '../../util'
import {renderHeader} from './header'
import {renderImage, renderFigcaption} from './image'
import {renderBacklinks} from './etc'
import {renderBody} from './body'
import {PostData, ThreadData} from '../models'

// Render post HTML
export default function (data: PostData|ThreadData): string {
	let cls = 'glass'
	if (data.editing) {
		cls += ' editing'
	}
	return HTML
		`<article id="p${data.id.toString()}" class="${cls}">
			${renderHeader(data)}
			${data.image ? renderFigcaption(data.image): ''}
			<div class="post-container">
				${data.image ? renderImage(data.image): ''}
				<blockquote>
					${renderBody(data)}
				</blockquote>
			</div>
			<small>
				${renderBacklinks(data.backlinks)}
			</small>
			${(data as any).subject ? renderOmit(data as ThreadData) : ""}
		</article>`
}

export function renderOmit(data: ThreadData): string {
	return ""
}
