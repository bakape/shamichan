// Miscellaneous helpers and event listeners

import {on} from './util'
import {write, threads} from './render'

export default () =>
	on(threads, "click", toggleSpoiler, {selector: "del"})

// Toggle spoiler revealing on click
function toggleSpoiler (event: Event) {
	write(() =>
		(event.target as Element).classList.toggle("reveal"))
}
