import {escape, pluralize, pad, makeAttrs} from '../../util'
import {renderImage, renderFigcaption} from './image'
import {renderBacklinks} from './etc'
import {renderBody} from './body'
import {PostData, ThreadData} from '../models'
import {posts as lang, time as timeLang} from '../../lang'
import options from '../../options'
import {PostCredentials} from "../posting/identity"

// Populate post template
export default function (frag: DocumentFragment, data: PostData|ThreadData) {
	if ((data as ThreadData).subject) {
		const el = frag.querySelector("h3")
		el.innerHTML = `「${escape((data as ThreadData).subject)}」`
		el.hidden = false
	}

	frag.querySelector("blockquote").innerHTML = renderBody(data)
	frag.querySelector("small").innerHTML = renderBacklinks(data.backlinks)

	renderHeader(frag, data)

	if (data.image) {
		renderFigcaption(frag.querySelector("figcaption"), data.image)
		renderImage(frag.querySelector("figure"), data.image)
	}
}

// Render the header on top of the post
export function renderHeader(frag: NodeSelector, data: PostData) {
	renderTime(frag.querySelector("time"), data.time)
	renderName(frag.querySelector(".name"), data)

	const nav = frag.querySelector("nav"),
		link = nav.firstElementChild as HTMLAnchorElement,
		qoute = nav.lastElementChild as HTMLAnchorElement
	link.href = `#p${data.id}`
	qoute.textContent = data.id.toString()
}

// Render the name of a post's poster
export function renderName(
	el: Element,
	{trip, name, auth, email}: PostCredentials,
) {
	if (options.anonymise) {
		el.innerHTML = lang.anon
		return
	}
	let html = ""

	if (email) {
		const attrs = {
			class: "email",
			href: "mailto:" + encodeURI(email),
			target: "_blank",
		}
		html += `<a ${makeAttrs(attrs)}>`
	}

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
		html += `<code>!${escape(trip)}</code>`
	}
	if (email) {
		html += "</a>"
	}
	if (auth) { // Render staff title
		el.classList.add("admin")
		html += ` ## ${escape(auth)}`
	}
	el.innerHTML = html
}

// TODO: Resolve, once moderation implemented
// // Renders a poster identification mnemonic
// export function renderMnemonic(mnemonic) {
// 	return `<b class="mod addr">${mnem}</b>`
// }

// Renders a time element. Can be either absolute or relative.
export function renderTime(el: Element, time: number) {
	// Format according to client's relative post timestamp setting
	let text = readableTime(time)
	if (options.relativeTime) {
		el.setAttribute("title", text)
		text = relativeTime(time, Math.floor(Date.now() / 1000))
	}
	el.textContent = text
}

// Renders classic absolute timestamp
function readableTime(time: number): string {
	let d = new Date(time * 1000)
	return `${pad(d.getDate())} ${timeLang.calendar[d.getMonth()]} `
		+ `${d.getFullYear()} (${timeLang.week[d.getDay()]}) `
		+`${pad(d.getHours())}:${pad(d.getMinutes())}`
}

// Renders readable elapsed time since post
function relativeTime(then: number, now: number): string {
	let time = Math.floor((now - then) / 60),
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
