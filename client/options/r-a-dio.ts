// R/a/dio integration

import { HTML, makeAttrs, fetchJSON } from '../util'
import options from '.'
import lang from '../lang'

type RadioData = {
	np: string
	listeners: number
	dj: string
	[index: string]: string | number
}

let el = document.getElementById('banner-center'),
	data: RadioData = {} as RadioData,
	started = false

// Fetch JSON from R/a/dio's API and rerender the banner, if different data
// received
async function fetchData() {
	const [res, err] = await fetchJSON<any>('https://r-a-d.io/api')
	if (err) {
		return console.warn(err)
	}
	const {
		main: {
			np,
			listeners,
			dj: {
				djname: dj,
			},
		},
	}
		= res

	const newData: RadioData = { np, listeners, dj }
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
		`<a href="http://r-a-d.io/" target="_blank">
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
