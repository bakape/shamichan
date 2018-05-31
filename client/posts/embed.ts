import { makeAttrs, makeFrag, escape, on, fetchJSON } from "../util"

type OEmbedDoc = {
	title?: string
	html?: string
	error?: string
}

// Types of different embeds by provider
enum provider { YouTube, SoundCloud, Vimeo, Coub, HookTube }

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
	[
		provider.HookTube,
		/https?:\/\/(?:[^\.]+\.)?(?:hooktube.com\/|hooktube.com\/embed\/|hooktube\.com\/watch\?v=)[a-zA-Z0-9_-]+/,
	],
]

// Map of providers to formatter functions
const formatters: { [key: number]: (s: string) => string } = {}

// Map of providers to information fetcher functions
const fetchers: { [key: number]: (el: Element) => Promise<void> } = {}

for (let p of ["YouTube", "SoundCloud", "Vimeo", "Coub", "HookTube"]) {
	const id = (provider as any)[p] as number
	formatters[id] = formatProvider(id)
	switch (id) {
		case provider.YouTube:
			fetchers[id] = proxyYoutube
			break
		case provider.HookTube:
			fetchers[id] = fetchHookTube
			break
		default:
			fetchers[id] = fetchNoEmbed(id)
	}
}

// formatter for the noembed.com meta-provider or hooktube
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

// Proxy all youtube requests to hooktube
function proxyYoutube(el: Element): Promise<void> {
	el.setAttribute("href", el.getAttribute("href")
		.replace("youtube", "hooktube")
		.replace("youtu.be", "hooktube.com"))
	return fetchHookTube(el)
}

// fetcher for the HookTube provider
async function fetchHookTube(el: Element): Promise<void> {
	// Use a CORS proxy to work around javascript's cross-domain restrictions
	const ref = el.getAttribute("href"),
		url = "https://hooktube.com/embed/" + ref
			.split(".com/").pop()
			.split("watch?v=").pop()
			.split("embed/").pop()
			.split("&").shift()
			.split("#").shift()
			.split("?").shift(),
		[data, err] = await fetchJSON<any>("https://cors-proxy.htmldriven.com/?url=" + url),
		time = !ref.includes("t=") ? "" : "&t=" + ref
			.split("t=").pop()
			.split("&").shift()
			.split("#").shift()
			.split("?").shift()

	if (err) {
		el.textContent = format(err, provider.HookTube)
		el.classList.add("errored")
		console.error(err)
		return
	}

	if (data.error) {
		el.textContent = format(data.error, provider.HookTube)
		el.classList.add("errored")
		return
	}

	el.textContent = format(new DOMParser()
		.parseFromString(data.body, "text/html").
		querySelectorAll('title')[0]
		.innerText, provider.HookTube)
	el.setAttribute("data-html",
		encodeURIComponent(
			`<iframe width="480" height="270" src="${url}?autoplay=false${time}" frameborder="0" allowfullscreen></iframe>`))
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

