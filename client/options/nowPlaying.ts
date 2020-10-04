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

type RadioRecord = {
	urlBase: string
	urlPath: string
	data: RadioData
	unmarshalfn: Function	// Unmarshal JSON response into RadioData object
}

// Entries must be in the same order as their option buttons
const radios: RadioRecord[] = [
	{
		urlBase: 'https://r-a-d.io/',
		urlPath: 'api',
		data: {} as RadioData,
		unmarshalfn: (res: any) => {
			const {
				main: {
					np, listeners,
					dj: {
						djname: dj,
					},
				},
			} = res;
			return { np, listeners, dj } as RadioData;
		},
	} as RadioRecord,
	{
		urlBase: 'https://www.edenofthewest.com/',
		urlPath: 'api/live/nowplaying/eden_radio',
		data: {} as RadioData,
		unmarshalfn: (res: any) => {
			const {
				listeners: {
					current: listeners,
				},
				live: {
					streamer_name: dj,
				},
				now_playing: {
					song: {
						text: np,
					},
				},
			} = res;
			return { np, listeners, dj } as RadioData;
		},
	} as RadioRecord,
];

let el = document.getElementById('banner-center'),
	started = false;

// Replacement new post names based on currently playing song
export const posterName = () =>
	_posterName;
let _posterName = "";
const songMap = new Map([
	[/Girls,? Be Ambitious/i, 'Joe'],
	[/Super Special/i, 'Super Special'],
]);

// Fetch JSON from enabled radio stations' API and rerender the banner, if different
// data received
async function fetchData(refresh: boolean = false) {
	let enabled: number[] = [];
	let changed = false;
	for (let i = 0; i < radios.length; i++) {
		// If station enabled
		if ((1 << i) & options.nowPlaying) {
			enabled.push(i);
			const [res, err] = await fetchJSON<any>(radios[i].urlBase + radios[i].urlPath);
			if (err) {
				console.warn(err);
				continue;
			}
			let newData = radios[i].unmarshalfn(res);
			if (!isMatch(newData, radios[i].data)) {
				changed = true;
				radios[i].data = newData;
			}
		}
	}
	if (changed || refresh) {
		render(enabled);
	}
}

// Compares all keys on a with keys on b for equality
function isMatch(a: {}, b: {}): boolean {
	for (let key in a) {
		if (a[key] !== b[key]) {
			return false;
		}
	}
	return true;
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
function render(enabled: number[] = []) {
	if (options.nowPlaying == 0) {
		el.innerHTML = _posterName = "";
		return;
	}

	let matched = false;
	let s: string[] = [];
	for (const i of enabled) {
		// Check for song matches
		if (!matched) {
			for (let [patt, rep] of songMap) {
				if (patt.test(radios[i].data.np)) {
					matched = true;
					_posterName = rep;
					break;
				}
			}
		}
		s.push(HTML
		`<div class="stream-info spaced">
			<a href="${radios[i].urlBase}" target="_blank">
				[${escape(radios[i].data.listeners.toString())}] ${escape(radios[i].data.dj)}
			</a>
			<a ${genAttrs(radios[i].data)}>
				<b>
					${escape(radios[i].data.np)}
				</b>
			</a>
		</div>`);
	}
	if (!matched) {
		_posterName = "";
	}
	el.innerHTML = s.join("");
}

// Initialize
export default function () {
	if (started) {
		return;
	}
	started = true;
	fetchData();

	// Handle toggling of the option
	let timer = setInterval(fetchData, 10000);
	options.onChange("nowPlaying", selection => {
		clearInterval(timer);
		if (selection == 0) {
			render();
		} else {
			timer = setInterval(fetchData, 10000);
			// Force rerender even if all RadioData is the same
			fetchData(true);
		}
	});
}
