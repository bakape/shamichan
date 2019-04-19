// R/a/dio  and Eden integration

import { HTML, makeAttrs, fetchJSON, escape } from '../util'
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
	started = false,
	dataEden: RadioData = {} as RadioData
// Replacement new post names based on currently playing song
export const posterName = () =>
	_posterName
let _posterName = ""
const songMap = new Map([
	[/Girls,? Be Ambitious/i, 'Joe'],
	[/Super Special/i, 'Super Special'],
])

// Fetch JSON from R/a/dio's or Eden's API and rerender the banner, if different data
// received
function radioData(res: any): RadioData {
	const {
		main: {
			np, listeners,
			dj: {
				djname: dj,
			},
		},
	} = res
	return { np, listeners, dj } as RadioData
}

function edenData(res: any): RadioData {
	const {
		dj: dj,
		current: np,
		listeners: listeners

	} = res
	return { np, listeners, dj } as RadioData
}

async function fetchData() {
	let newData = {} as RadioData
	if (options.nowPlaying === "r/a/dio") {
		const [res, err] = await fetchJSON<any>('https://r-a-d.io/api')
		if (err) {
			return console.warn(err)
		}

		newData = radioData(res)
	} else if (options.nowPlaying === "eden") {
		const [res, err] = await fetchJSON<any>('https://edenofthewest.com/ajax/status.php')
		if (err) {
			return console.warn(err)
		}

		newData = edenData(res)
	}
	else if (options.nowPlaying === "both") {
		let newDataEden = {} as RadioData
		const [res, err] = await fetchJSON<any>('https://r-a-d.io/api')
		const [resEden, errEden] = await fetchJSON<any>('https://edenofthewest.com/ajax/status.php')
		if (err) {
			return console.warn(err)
		}
		if (errEden) {
			return console.warn(errEden)
		}

		newData = radioData(res)
		newDataEden = edenData(resEden)

		data = newData
		dataEden = newDataEden
		render()
	}

	if (!isMatch(newData, data) && (options.nowPlaying != "both")) {
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
	if (options.nowPlaying === "none") {
		el.innerHTML = _posterName = ""
		return
	}

	// Check for song matches
	let matched = false
	for (let [patt, rep] of songMap) {
		if (patt.test(data.np)) {
			matched = true
			_posterName = rep
			break
		}
	}
	if (!matched) {
		_posterName = ""
	}

	if (options.nowPlaying === "both") {
		const attrsRadio = {
			title: lang.ui["googleSong"],
			href: `https://google.com/search?q=${encodeURIComponent(data.np)}`,
			target: "_blank",
		}
		const attrsEden = {
			title: lang.ui["googleSong"],
			href: `https://google.com/search?q=${encodeURIComponent(dataEden.np)}`,
			target: "_blank",
		}
		el.innerHTML = HTML
			`<a href="https://r-a-d.io/" target="_blank">
				[${escape(data.listeners.toString())}] ${escape(data.dj)}
			</a>
			&nbsp;&nbsp;
			<a ${makeAttrs(attrsRadio)}>
				<b>
					${escape(data.np)}
				</b>
			</a>
			 |
			<a href="https://edenofthewest.com/" target="_blank">
				[${escape(dataEden.listeners.toString())}] ${escape(dataEden.dj)}
			</a>
			&nbsp;&nbsp;
			<a ${makeAttrs(attrsEden)}>
				<b>
					${escape(dataEden.np)}
				</b>
			</a>`


	}
	else {
		const attrs = {
			title: lang.ui["googleSong"],
			href: `https://google.com/search?q=${encodeURIComponent(data.np)}`,
			target: "_blank",
		}
		const site = options.nowPlaying === "eden" ? "edenofthewest.com" : "r-a-d.io"
		el.innerHTML = HTML
			`<a href="https://${site}/" target="_blank">
			[${escape(data.listeners.toString())}] ${escape(data.dj)}
		</a>
		&nbsp;&nbsp;
		<a ${makeAttrs(attrs)}>
			<b>
				${escape(data.np)}
			</b>
		</a>`
	}
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
	options.onChange("nowPlaying", selection => {
		if (selection === "none") {
			clearInterval(timer)
			render()
		} else {
			timer = setInterval(fetchData, 10000)
			fetchData()
		}
	})
}
