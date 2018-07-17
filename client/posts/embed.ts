import { makeAttrs, makeFrag, escape, on, fetchJSON } from "../util"

type OEmbedDoc = {
	title?: string
	html?: string
	error?: string
}

// Types of different embeds by provider
enum provider { YouTube, SoundCloud, Vimeo, Coub }

// Matching patterns and their respective providers
const patterns: [provider, RegExp][] = [
	[
		provider.YouTube,
		/https?:\/\/(?:[^\.]+\.)?(?:youtu\.be\/|youtube\.com\/embed\/|youtube\.com\/watch\?v=)[a-zA-Z0-9_-]+/,
	],
	[
		provider.SoundCloud,
		/https?:\/\/soundcloud.com\/.*/,
	],
	[
		provider.Vimeo,
		/https?:\/\/(?:www\.)?vimeo\.com\/.+/,
	],
	[
		provider.Coub,
		/https?:\/\/(?:www\.)?coub\.com\/view\/.+/,
	],
]

// Map of providers to formatter functions
const formatters: { [key: number]: (s: string) => string } = {}

// Map of providers to information fetcher functions
const fetchers: { [key: number]: (el: Element) => Promise<void> } = {}

for (let p of ["YouTube", "SoundCloud", "Vimeo", "Coub"]) {
	const id = (provider as any)[p] as number
	formatters[id] = formatProvider(id)
	switch (id) {
		case provider.YouTube:
			fetchers[id] = fetchYouTube()
			break
		default:
			fetchers[id] = fetchNoEmbed(id)
	}
}

// formatter for the noembed.com meta-provider or YouTube
function formatProvider(type: provider): (s: string) => string {
	return (href: string) => {
		const attrs = {
			rel: "noreferrer",
			href: escape(href),
			class: "embed",
			target: "_blank",
			"data-type": type.toString(),
		}
		return `<em><a ${makeAttrs(attrs)}>[${provider[type]}] ???</a></em>`
	}
}

// fetcher for the YouTube provider
function fetchYouTube(): (el: Element) => Promise<void> {
	return async (el: Element) => {
		const ref = el.getAttribute("href"),
		id = strip(ref.split(".be/").pop().split("embed/").pop().split("watch?v=")),
		res = await fetch("/api/get-youtube-data/" + id),
		[title, video] = (await res.text()).split("\n")

		switch (res.status) {
		case 200:
			el.textContent = format(title, provider.YouTube)
			break
		case 500:
			el.textContent = format("Error 500: YouTube is not available", provider.YouTube)
			el.classList.add("errored")
			return
		default:
			const errmsg = `Error ${res.status}: ${res.statusText}`
			el.textContent = format(errmsg, provider.YouTube)
			el.classList.add("errored")
			console.error(errmsg)
			return
		}

		if (!title) {
			el.textContent = format("Error: Title does not exist", provider.YouTube)
			el.classList.add("errored")
			return
		}

		if (!video) {
			el.textContent = format("Error: Empty googlevideo URL", provider.YouTube)
			el.classList.add("errored")
			return
		}

		el.setAttribute("data-html", encodeURIComponent(
			`<video width="480" height="270" ` + (ref.includes("loop=1") ? "loop " : '') + `controls>`
			+ `<source src="` + video + (!ref.includes(`t=`) ? check("start") : '') + check("t") + `" type="video/mp4">`
			+ `Your browser does not support the video tag.`
			+ `</video>`))
			
		function strip(s: string[]): string {
			return s.pop().split('&').shift().split('#').shift().split('?').shift()
		}

		function check(s: string): string {
			return ref.includes(`${s}=`) ? `#t=` + strip(ref.split(`${s}=`)) : ''
		}
	}
}

// fetcher for the noembed.com meta-provider
function fetchNoEmbed(type: provider): (el: Element) => Promise<void> {
	return async (el: Element) => {
		const url = "https://noembed.com/embed?url=" + el.getAttribute("href"),
			[data, err] = await fetchJSON<OEmbedDoc>(url)

		if (err) {
			el.textContent = format(err, type)
			el.classList.add("errored")
			console.error(err)
			return
		}

		if (data.error) {
			el.textContent = format(data.error, type)
			el.classList.add("errored")
			return
		}

		el.textContent = format(data.title, type)
		el.setAttribute("data-html", encodeURIComponent(data.html.trim()))
	}
}

function format(s: string, type: provider): string {
	return `[${provider[type]}] ${s}`
}

// Match and parse URL against embeddable formats. If matched, returns the
// generated HTML embed string, otherwise returns empty string.
export function parseEmbeds(s: string): string {
	for (let [type, patt] of patterns) {
		if (patt.test(s)) {
			return formatters[type](s)
		}
	}
	return ""
}

// Fetch and render any metadata int the embed on mouseover
function fetchMeta(e: MouseEvent) {
	const el = e.target as Element
	if (el.hasAttribute("data-title-requested")
		|| el.classList.contains("expanded")
	) {
		return
	}
	el.setAttribute("data-title-requested", "true")
	execFetcher(el)
}

function execFetcher(el: Element): Promise<void> {
	return fetchers[parseInt(el.getAttribute("data-type"))](el)
}

// Toggle the expansion of an embed
async function toggleExpansion(e: MouseEvent) {
	const el = e.target as Element

	// Don't trigger, when user is trying to open in a new tab or fetch has
	// errored
	if (e.which !== 1 || e.ctrlKey || el.classList.contains("errored")) {
		return
	}
	e.preventDefault()

	if (el.classList.contains("expanded")) {
		el.classList.remove("expanded")
		const iframe = el.lastChild
		if (iframe) {
			iframe.remove()
		}
		return
	}

	// The embed was clicked before a mouseover (ex: touch screen)
	if (!el.hasAttribute("data-html")) {
		await execFetcher(el)
	}

	const html = decodeURIComponent(el.getAttribute("data-html")),
		frag = makeFrag(html)

	// Restrict embedded iframe access to the page. Improves privacy.
	for (let el of frag.querySelectorAll("iframe")) {
		el.setAttribute("referrerpolicy", "no-referrer")
		el.setAttribute(
			"sandbox",
			"allow-scripts allow-same-origin allow-popups allow-modals",
		)
	}

	el.append(frag)
	el.classList.add("expanded")
}

on(document, "mouseover", fetchMeta, {
	passive: true,
	selector: "a.embed",
})
on(document, "click", toggleExpansion, {
	selector: "a.embed",
})

