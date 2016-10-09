import { makeAttrs, makeFrag } from "../util"
import { $threads, write } from "../render"
import { on } from "../util"
import { fetchJSON } from "../json"

type OEmbedDoc = {
	title: string
	html: string
}

// Types of different embeds by provider
enum provider { Youtube, SoundCloud, Vimeo, Twitch }

// Matching patterns and their respective providers
const patterns: [provider, RegExp][] = [
	[
		provider.Youtube,
		/https?:\/\/(?:[^\.]+\.)?youtube\.com\/watch\/?\?(?:.+&)?v=([^&]+)/,
	],
	[
		provider.Youtube,
		/https?:\/\/(?:[^\.]+\.)?(?:youtu\.be|youtube\.com\/embed)\/([a-zA-Z0-9_-]+)/,
	],
	[
		provider.SoundCloud,
		/https?:\/\/soundcloud.com\/.*/,
	],
	[
		provider.Vimeo,
		/https?:\/\/(?:www\.)?vimeo\.com\/.+/,
	],
]

// Map of providers to formatter functions
const formatters: { [key: number]: (s: string) => string } = {}

// Map of providers to information fetcher functions
const fetchers: { [key: number]: (el: Element) => Promise<void> } = {}

for (let p of ["Youtube", "SoundCloud", "Vimeo"]) {
	const id = (provider as any)[p] as number
	formatters[id] = formatNoEmbed(id)
	fetchers[id] = fetchNoEmbed
}

// Formatter for the noembed.com meta-provider
function formatNoEmbed(type: provider): (s: string) => string {
	return (href: string) => {
		const attrs = {
			href,
			class: "embed",
			target: "_blank",
			"data-type": type.toString(),
		}
		return `<em><a ${makeAttrs(attrs)}>${provider[type]} ???</a></em>`
	}
}

// fetcher for the noembed.com meta-provider
async function fetchNoEmbed(el: Element) {
	const url = "https://noembed.com/embed?url="
		+ encodeURI(el.getAttribute("href"))
	const {title, html} = await fetchJSON<OEmbedDoc>(url)

	el.textContent = title
	el.setAttribute("data-html", encodeURIComponent(html.trim()))
}

// Match and parse URL against embedable formats. If matched, returns the
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
	// Don't trigger, when user is trying to open in a new tab
	if (e.which !== 1 || e.ctrlKey) {
		return
	}
	e.preventDefault()
	const el = e.target as Element

	if (el.classList.contains("expanded")) {
		write(() => {
			el.classList.remove("expanded")
			const iframe = el.lastChild
			if (iframe) {
				iframe.remove()
			}
		})
		return
	}

	// Somehow the embed was clicked before a mouseover
	if (!el.hasAttribute("data-html")) {
		await execFetcher(el)
	}

	const html = decodeURIComponent(el.getAttribute("data-html")),
		frag = makeFrag(html)
	write(() => {
		el.append(frag)
		el.classList.add("expanded")
	})
}

on($threads, "mouseover", fetchMeta, {
	passive: true,
	selector: ".embed",
})

on($threads, "click", toggleExpansion, {
	selector: ".embed",
})

