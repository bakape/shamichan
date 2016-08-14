import {HTML} from '../../util'
import {renderHeader} from './header'
import {renderImage, renderFigcaption} from './image'
import {renderBacklinks} from './etc'
import {renderBody} from './body'
import {PostData, ThreadData} from '../models'

// Render post HTML contents
export default function (data: PostData|ThreadData): string {
	return HTML
		`${renderHeader(data)}
		${data.image ? renderFigcaption(data.image): ''}
		<div class="post-container">
			${data.image ? renderImage(data.image): ''}
			<blockquote>
				${renderBody(data)}
			</blockquote>
		</div>
		<small>
			${renderBacklinks(data.backlinks)}
		</small>`
}
