// eden integration

import { HTML, makeAttrs, fetchJSON } from '../util'
import options from '.'
import lang from '../lang'

type EdenData = {
	np: string
	listeners: number
	dj: string
	[index: string]: string | number
}

let el = document.getElementById('banner-center'),
	data: EdenData = {} as EdenData,
	started = false

// Fetch JSON from Eden's API and rerender the banner, if different data
// received
async function fetchData() {
	const [res, err] = await fetchJSON<any>('https://edenofthewest.com/ajax/status.php')
	if (err) {
		return console.warn(err)
	}
	const {
        dj: dj,
        current: np,
        listeners: listeners
	}
		= res

	const newData: EdenData = { np, listeners, dj }
	if (!isMatch(newData, data)) {
		data = newData
		render()
	}
}

// Compares all keys on a with keys on b for equality
function isMatch(a: {}, b: {}): boolean {
	for (let key in a) {
		if (a[key] !== b[key]) {
			return false
		}
	}
	return true
}

// Render the banner message text
function render() {
	if (!options.nowPlaying) {
		el.innerHTML = ""
		return
	}

	const attrs = {
		title: lang.ui["googleSong"],
		href: `https://google.com/search?q=${encodeURIComponent(data.np)}`,
		target: "_blank",
	}
	el.innerHTML = HTML
		`<a href="https://edenofthewest.com/" target="_blank">
			[${data.listeners.toString()}] ${data.dj}
		</a>
		&nbsp;&nbsp;
		<a ${makeAttrs(attrs)}>
			<b>
				${data.np}
			</b>
		</a>`
}

// Initialize
export default function () {
	if (started) {
		return
	}
	started = true
	fetchData()

	// Handle toggling of the option
	let timer = setInterval(fetchData, 10000)
	options.onChange("nowPlaying", enabled => {
		if (!enabled) {
			clearInterval(timer)
			render()
		} else {
			timer = setInterval(fetchData, 10000)
			fetchData()
		}
	})
}
