import { escape, pluralize, pad } from '../../util'
import { renderImage } from './image'
import { renderBacklinks } from './etc'
import renderBody from './body'
import { PostData, ThreadData } from '../../common'
import lang from '../../lang'
import options from '../../options'
import { page } from "../../state"

interface PostCredentials {
	name?: string
	trip?: string
	auth?: string
}

// Populate post template
export default function (post: Element, data: PostData | ThreadData) {
	if ((data as ThreadData).subject) {
		const el = post.querySelector("h3")
		el.innerHTML = `「${escape((data as ThreadData).subject)}」`
		el.hidden = false
	}

	post.querySelector("blockquote").innerHTML = renderBody(data)
	renderBacklinks(post, data.backlinks)
	if (data.banned) {
		renderBanned(post)
	}
	renderHeader(post, data)
	if (data.image) {
		renderImage(post, data.image, false)
	}
}

// Render "USER WAS BANNED FOR THIS POST" message
export function renderBanned(parent: NodeSelector) {
	if (parent.querySelector(".banned")) {
		return
	}
	const b = document.createElement("b")
	b.classList.add("admin", "banned")
	b.innerText = lang.posts["banned"]
	parent.querySelector("blockquote").after(b)
}

// Render the header on top of the post
export function renderHeader(frag: NodeSelector, data: PostData) {
	renderTime(frag.querySelector("time"), data.time, false)
	renderName(frag.querySelector(".name"), data)

	const nav = frag.querySelector("nav"),
		link = nav.firstElementChild as HTMLAnchorElement,
		quote = nav.lastElementChild as HTMLAnchorElement
	let url = `${data.op || data.id}#p${data.id}`
	if (page.thread) {
		url = "/all/" + url
	}
	quote.href = link.href = url
	quote.textContent = data.id.toString()
}

// Render the name of a post's poster
export function renderName(
	el: Element,
	{trip, name, auth}: PostCredentials,
) {
	if (options.anonymise) {
		el.innerHTML = lang.posts["anon"]
		return
	}
	let html = ""

	if (name || !trip) {
		if (name) {
			html += escape(name)
		} else {
			html += lang.posts["anon"]
		}
		if (trip) {
			html += ' '
		}
	}

	if (trip) {
		html += `<code>!${escape(trip)}</code>`
	}
	if (auth) { // Render staff title
		el.classList.add("admin")
		html += ` ## ${lang.posts[auth] || "??"}`
	}
	el.innerHTML = html
}

// TODO: Resolve, once moderation implemented
// // Renders a poster identification mnemonic
// export function renderMnemonic(mnemonic) {
// 	return `<b class="mod addr">${mnemonic}</b>`
// }

// Renders a time element. Can be either absolute or relative.
export function renderTime(el: Element, time: number, forceRelative: boolean) {
	// Format according to client's relative post timestamp setting
	let text = readableTime(time)
	if (forceRelative || options.relativeTime) {
		el.setAttribute("title", text)
		text = relativeTime(time, Math.floor(Date.now() / 1000))
	}
	el.textContent = text
}

// Renders classic absolute timestamp
function readableTime(time: number): string {
	let d = new Date(time * 1000)
	return `${pad(d.getDate())} ${lang.time.calendar[d.getMonth()]} `
		+ `${d.getFullYear()} (${lang.time.week[d.getDay()]}) `
		+ `${pad(d.getHours())}:${pad(d.getMinutes())}`
}

// Renders readable elapsed time since post
function relativeTime(then: number, now: number): string {
	let time = Math.floor((now - then) / 60),
		isFuture = false
	if (time < 1) {
		if (time > -5) { // Assume to be client clock imprecision
			return lang.posts["justNow"]
		} else {
			isFuture = true
			time = -time
		}
	}

	const divide = [60, 24, 30, 12],
		unit = ['minute', 'hour', 'day', 'month']
	for (let i = 0; i < divide.length; i++) {
		if (time < divide[i]) {
			return ago(time, lang.plurals[unit[i]], isFuture)
		}
		time = Math.floor(time / divide[i])
	}

	return ago(time, lang.plurals["year"], isFuture)
}

// Renders "56 minutes ago" or "in 56 minutes" like relative time text
function ago(time: number, units: [string, string], isFuture: boolean): string {
	const count = pluralize(time, units)
	if (isFuture) {
		return `${lang.posts["in"]} ${count}`
	}
	return `${count} ${lang.posts["ago"]}`
}
