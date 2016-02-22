import {escape} from 'underscore'
import {parseHTML, parseAttributes, pad} from '../../util'
import lang from 'lang'
import {config} from '../../state'
import options from '../../options'

// Render the header with various post information
export function renderHeader(data) {
	const {num, op, subject} = data,
		postURL = renderPostURL(num)
	return parseHTML
		`<header>
			<input type="checkbox" class="postCheckbox">
			${subject ? `<h3>「${escape(data.subject)}」</h3>` : ''}
			${renderName(data)}
			${renderTime(data.time)}
			<nav>
				<a href="${postURL}" class="history">
					No.
				</a>
				<a href="${postURL}" class="quote">
					${num}
				</a>
			</nav>
		</header>
		<span class="oi control" data-glyph="chevron-bottom"></span>`
}

// Render the name of a post's poster
export function renderName(data) {
	let html = '<b class="name'
	const {auth, email} = data
	if (auth) {
		html += ` ${auth === 'admin' ? 'admin' : 'moderator'}`
	}
	html += '">'
	if (email) {
		const attrs = {
			class: 'email',
			href: 'mailto:' + encodeURI(email),
			target: 'blank'
		}
		html += `<a ${parseAttributes(attrs)}>`
	}
	html += resolveName(data)
	if (email) {
		html += '</a>'
	}
	html += '</b>'
	if (data.mnemonic) {
		html += ' ' + renderMnemonic(data.mnemonic)
	}
	return html
}

// Determine the name and tripcode combination to render
function resolveName(data) {
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
		let alias
		if (auth in config.staff.classes) {
			alias = config.staff.classes[auth].alias
		} else {
			alias = auth
		}
		html += ` ## ${alias}`
	}
	return html
}

// Renders a poster identification mnemonic
export function renderMnemonic(mnemonic) {
	return `<b class="mod addr">${mnem}</b>`
}

// Renders a time element. Can be either absolute or relative.
export function renderTime(time) {
	// Format according to client's relative post timestamp setting
	let title, text
	const readable = readableTime(time)
	if (options.get('relativeTime')) {
		title = readable
		text = relativeTime(time, Date.now())
	}
	return parseHTML
		`<time title="${title}">
			${text || readable}
		</time>`
}

// Renders classic absolute timestamp
function readableTime(time) {
	let d = new Date(time)
	return pad(d.getDate()) + ' '
		+ lang.time.year[d.getMonth()] + ' '
		+ d.getFullYear()
		+ `(${lang.time.week[d.getDay()]})`
		+`${pad(d.getHours())}:${pad(d.getMinutes())}`
}

// Renders readable elapsed time since post
function relativeTime(then, now) {
	let time = Math.floor((now - then) / 60000),
		isFuture = false
	if (time < 1) {
		if (time > -5) { // Assume to be client clock imprecission
			return lang.time.just_now
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
			return lang.ago(time, lang.time[unit[i]], isFuture)
		}
		time = Math.floor(time / divide[i])
	}

	return lang.ago(time, lang.time.year, isFuture)
}

// Render an anchor that points to the target post number
export function renderPostURL(num) {
	return `#p${num}`
}
