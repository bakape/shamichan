import {HTML, makeAttrs, pad, escape} from '../../util'
import {config} from '../../state'
import options from '../../options'
import {PostData, ThreadData} from '../models'
import {posts as lang, time as timeLang} from '../../lang'

// Render the header with various post informationt
export function renderHeader(data: PostData): string {
	const {id, op, subject} = data as any,
		postURL = renderPostURL(id)
	let subjectHTML: string
	return HTML
		`<header>
			${subject ? `<h3>「${escape(subject)}」</h3>` : ''}
			${renderName(data)}
			${renderTime(data.time)}
			<nav>
				<a href="${postURL}" class="history">
					No.
				</a>
				<a href="${postURL}" class="quote">
					${id.toString()}
				</a>
			</nav>
			<span class="control">
				<a>
					⮟
				</a>
			</span>
		</header>`
}

// Render the name of a post's poster
export function renderName(data: PostData): string {
	let html = '<b class="name'
	const {auth, email} = data
	if (auth) {
		html += ` ${auth === 'admin' ? 'admin' : 'moderator'}`
	}
	html += '">'
	if (email) {
		const attrs: StringMap = {
			class: 'email',
			href: 'mailto:' + encodeURI(email),
			target: 'blank'
		}
		html += `<a ${makeAttrs(attrs)}>`
	}
	html += resolveName(data)
	if (email) {
		html += '</a>'
	}
	html += '</b>'
	return html
}

// Determine the name and tripcode combination to render
function resolveName(data: PostData): string {
	let html = ''
	const {trip, name, auth} = data

	if (name || !trip) {
		if (name) {
			html += escape(name)
		} else {
			html += lang.anon
		}
		if (trip) {
			html += ' '
		}
	}

	if (trip) {
		html += `<code>${escape(trip)}</code>`
	}

	if (auth) { // Render staff title
		html += ` ## ${auth}`
	}

	return html
}

// TODO: Resolve, once moderation implemented
// // Renders a poster identification mnemonic
// export function renderMnemonic(mnemonic) {
// 	return `<b class="mod addr">${mnem}</b>`
// }

// Renders a time element. Can be either absolute or relative.
export function renderTime(time: number): string {
	// Format according to client's relative post timestamp setting
	let title: string,
		text :string
	const readable = readableTime(time)
	if (options.relativeTime) {
		title = readable
		text = relativeTime(time, Date.now())
	}

	let html = "<time"
	if (title) {
		html += ` title="${title}"`
	}
	html += `>${text || readable}</time>`
	return html
}

// Renders classic absolute timestamp
function readableTime(time: number): string {
	let d = new Date(time)
	return pad(d.getDate()) + ' '
		+ timeLang.calendar[d.getMonth()] + ' '
		+ d.getFullYear()
		+ `(${timeLang.week[d.getDay()]})`
		+`${pad(d.getHours())}:${pad(d.getMinutes())}`
}

// Renders readable elapsed time since post
function relativeTime(then: number, now: number): string {
	let time = Math.floor((now - then) / 60000),
		isFuture = false
	if (time < 1) {
		if (time > -5) { // Assume to be client clock imprecission
			return timeLang.justNow
		}
		else {
			isFuture = true
			time = -time
		}
	}

	const divide = [60, 24, 30, 12],
		unit = ['minute', 'hour', 'day', 'month']
	for (let i = 0; i < divide.length; i++) {
		if (time < divide[i]) {
			return ago(time, timeLang[unit[i]] as string[], isFuture)
		}
		time = Math.floor(time / divide[i])
	}

	return ago(time, timeLang.year, isFuture)
}

// Renders "56 minutes ago" or "in 56 minutes" like relative time text
function ago(time: number, units: string[], isFuture: boolean): string {
	const unit = units[time > 1 ? 1 : 0]
	let text: string
	if (isFuture) {
		text = `${timeLang.in} ${time} ${unit}`
	} else {
		text = `${time} ${unit} ${timeLang.ago}`
	}
	return text
}

// Render an anchor that points to the target post number
export function renderPostURL(id: number): string {
	return `#p${id}`
}
