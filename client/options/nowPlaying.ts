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
	dataEden: RadioData = {} as RadioData;

// Replacement new post names based on currently playing song
export const posterName = () =>
	_posterName
let _posterName = ""
const songMap = new Map([
	[/Girls,? Be Ambitious/i, 'Joe'],
	[/Super Special/i, 'Super Special'],
])

// Fetch JSON from R/a/dio's or Eden's API and rerender the banner, if different
// data received
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
		listeners: {
			current: listeners,
		},
		now_playing: {
			streamer: dj,
			song: {
				text: np,
			},
		},
	} = res
	return { np, listeners, dj } as RadioData
}

async function fetchData() {
	const radioURL = 'https://r-a-d.io/api';
	const edenURL = 'https://www.edenofthewest.com/api/live/nowplaying/eden_radio';
	let newData = {} as RadioData;
	switch (options.nowPlaying) {
		case "r/a/dio":
			{
				const [res, err] = await fetchJSON<any>(radioURL);
				if (err) {
					return console.warn(err);
				}

				newData = radioData(res);
			}
			break;
		case "eden":
			{
				const [res, err] = await fetchJSON<any>(edenURL);
				if (err) {
					return console.warn(err);
				}

				newData = edenData(res);
			}
			break;
		case "both":
			{
				let newDataEden = {} as RadioData;
				const [res, err] = await fetchJSON<any>(radioURL);
				const [resEden, errEden] = await fetchJSON<any>(edenURL);
				if (err) {
					return console.warn(err);
				}
				if (errEden) {
					return console.warn(errEden);
				}

				newData = radioData(res);
				newDataEden = edenData(resEden);

				data = newData;
				dataEden = newDataEden;
				render();
			}
			break;
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

function genAttrs(data: RadioData): string {
	return makeAttrs({
		title: lang.ui["googleSong"],
		// Remove hyphens to prevent google from generating exclusions
		href: `https://google.com/search?q=`
			+ encodeURIComponent(data.np.replace(/\-/g, ' ')),
		target: "_blank",
	});
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
		el.innerHTML = HTML
			`<a href="https://r-a-d.io/" target="_blank">
				[${escape(data.listeners.toString())}] ${escape(data.dj)}
			</a>
			<a ${genAttrs(data)}>
				<b>
					${escape(data.np)}
				</b>
			</a>
			 |
			<a href="https://edenofthewest.com/" target="_blank">
				[${escape(dataEden.listeners.toString())}] ${escape(dataEden.dj)}
			</a>
			<a ${genAttrs(dataEden)}>
				<b>
					${escape(dataEden.np)}
				</b>
			</a>`
	}
	else {
		const site = options.nowPlaying === "eden"
			? "edenofthewest.com"
			: "r-a-d.io";
		el.innerHTML = HTML
			`<a href="https://${site}/" target="_blank">
			[${escape(data.listeners.toString())}] ${escape(data.dj)}
		</a>
		<a ${genAttrs(data)}>
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
