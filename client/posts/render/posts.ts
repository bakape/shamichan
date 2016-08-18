import {escape, pluralize, pad} from '../../util'
import {renderImage, renderFigcaption} from './image'
import {renderBacklinks} from './etc'
import {renderBody} from './body'
import {PostData, ThreadData} from '../models'
import {posts as lang, time as timeLang} from '../../lang'
import options from '../../options'

// Populate post template
export default function (frag: DocumentFragment, data: PostData|ThreadData) {
	if ((data as ThreadData).subject) {
		const el = frag.querySelector("h3")
		el.innerHTML = `「${escape((data as ThreadData).subject)}」`
		el.hidden = false
	}

	renderTime(frag.querySelector("time"), data.time)
	renderName(frag.querySelector(".name"), data)
	frag.querySelector("blockquote").innerHTML = renderBody(data)
	frag.querySelector("small").innerHTML = renderBacklinks(data.backlinks)

	const nav = frag.querySelector("nav"),
		link = nav.firstChild as HTMLAnchorElement,
		qoute = nav.lastChild as HTMLAnchorElement
	link.href = qoute.href = `#p${data.id}`
	qoute.textContent = data.id.toString()

	if (data.image) {
		renderFigcaption(frag.querySelector("figcaption"), data.image)
		renderImage(frag.querySelector("figure"), data.image)
	}
}

// Render the name of a post's poster
export function renderName(el: Element, data: PostData) {
	let text = ""
	const {trip, name, auth} = data

	if (name || !trip) {
		if (name) {
			text += escape(name)
		} else {
			text += lang.anon
		}
		if (trip) {
			text += ' '
		}
	}

	if (trip) {
		const code = el.lastElementChild
		code.hidden = false
		code.textContent = escape(trip)
	}
	if (auth) { // Render staff title
		el.classList.add("admin")
		text += ` ## ${auth}`
	}
	el.prepend(text)
}

// TODO: Resolve, once moderation implemented
// // Renders a poster identification mnemonic
// export function renderMnemonic(mnemonic) {
// 	return `<b class="mod addr">${mnem}</b>`
// }

// Renders a time element. Can be either absolute or relative.
export function renderTime(el: Element, time: number) {
	// Format according to client's relative post timestamp setting
	let title: string,
		text :string
	const readable = readableTime(time)
	if (options.relativeTime) {
		title = readable
		text = relativeTime(time, Date.now())
	}

	if (title) {
		el.setAttribute("title", title)
	}
	el.textContent = text
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
	const count = pluralize(time, units)
	return isFuture ? `${timeLang.in} ${count}` : `${count} ${timeLang.ago}`
}
