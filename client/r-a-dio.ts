// R/a/dio integration

import {HTML, makeAttrs, isMatch} from './util'
import {fetchJSON} from "./json"
import options from './options'
import {write} from './render'
import {banner as lang} from './lang'

type RadioData = {
	np: string
	listeners: number
	dj: string
	[index: string]: string|number
}

let el = document.getElementById('banner-center'),
	data: RadioData = {} as RadioData

// Fetch JSON from R/a/dio's API and rerender the banner, if different data
// received
async function fetchData() {
	const {
		main: {
			np,
			listeners,
			dj: {
				djname: dj,
			},
		},
	} =
		await fetchJSON<any>('https://r-a-d.io/api')

	const newData: RadioData = {np, listeners, dj}
	if (!isMatch(newData, data)) {
		data = newData
		render()
	}
}

// Render the banner message text
function render() {
	if (!options.nowPlaying) {
		write(() =>
			el.innerHTML = "")
		return
	}

	const attrs = {
		title: lang.googleSong,
		href: `https://google.com/search?q=${encodeURIComponent(data.np)}`,
		target: "_blank",
	}
	const html = HTML
		`<a href="http://r-a-d.io/" target="_blank">
			[${data.listeners.toString()}] ${data.dj}
		</a>
		&nbsp;&nbsp;
		<a ${makeAttrs(attrs)}>
			<b>
				${data.np}
			</b>
		</a>`
	write(() =>
		el.innerHTML = html)
}

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
