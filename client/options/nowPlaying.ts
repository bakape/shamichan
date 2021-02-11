import lang from "../lang"
import { HTML, escape } from "../util"

class RadioData {
	constructor(
		readonly listeners: number,
		readonly song: string,
		readonly streamer: string,
	) { }

	private static descriptors = [
		["listeners", "number"],
		["song", "string"],
		["streamer", "string"],
	] as const

	static is(data: unknown): data is RadioData {
		if (typeof data !== "object" || data === null) {
			return false
		}
		for (const [key, type] of RadioData.descriptors) {
			if (key in data && typeof (data as any)[key] === type) {
				continue
			}
			return false
		}
		return true
	}

	static equal(a: RadioData, b: RadioData) {
		for (const [key] of RadioData.descriptors) {
			if (a[key] === b[key]) {
				continue
			}
			return false
		}
		return true
	}
}

let globalPostName: string | undefined
export const getPostName = () => globalPostName

function postNameFor(song: string) {
	for (const [pattern, name] of [
		[/Girls,? Be Ambitious/i, "Joe"],
		[/Super Special/i, "Super Special"],
	] as const) {
		if (pattern.test(song)) {
			return name
		}
	}
}

class Banner {
	data?: RadioData
	postName?: string
	timer?: ReturnType<typeof setInterval>

	readonly element: HTMLElement

	constructor(
		readonly url: string,
		readonly api: string,
		readonly unmarshal: (data: any) => Record<keyof RadioData, unknown>,
	) {
		this.element = document.createElement("div")
		this.element.classList.add("spaced")
		document.getElementById("banner-center")!.append(this.element)
	}

	async fetch() {
		const response = await fetch(this.api)
		if (!response.ok) {
			throw Error(await response.text())
		}
		const data = this.unmarshal(await response.json())
		if (!RadioData.is(data)) {
			throw Error("Unexpected response")
		}
		return data
	}

	async update() {
		const data = await this.fetch()
		if (this.data && RadioData.equal(this.data, data)) {
			return
		}

		this.element.innerHTML = HTML`
		<a href="${this.url}" target="_blank">
			[${escape(data.listeners.toString())}] ${escape(data.streamer)}
		</a>
		<a href="https://google.com/search?q=${encodeURIComponent(data.song.replace(/-/g, " "))}" target="_blank" title="${lang.ui["googleSong"]}">
			<b>
				${escape(data.song)}
			</b>
		</a>`

		if (this.data?.song !== data.song) {
			if (globalPostName === undefined || globalPostName === this.postName) {
				globalPostName = this.postName = postNameFor(data.song)
			}
		}

		this.data = data
	}

	toggle(enabled: boolean) {
		this.element.classList.toggle("hidden", !enabled)
		clearInterval(this.timer)
		if (enabled) {
			const update = () => this.update().catch(console.warn)
			this.timer = setInterval(update, 10000)
			update()
		} else {
			delete this.data
			delete this.postName
			delete this.timer
		}
	}
}

export function toggle(...args: ConstructorParameters<typeof Banner>) {
	const banner = new Banner(...args)
	return (enabled = false) => banner.toggle(enabled)
}
