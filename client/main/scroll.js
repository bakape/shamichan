/*
 * Various page scrolling logic
 */

import {Backbone, options, state, $threads, $banner, events} from 'main'

let atBottom, lockIndicator, isThread

/**
 * Write the current state to variables
 */
function cacheState() {
	lockIndicator = document.query('#lock')
	isThread = !!state.posts.get('thread')
}

// Recache state on page change
state.page.on('change', cacheState)
cacheState()

/**
 * Set the scroll lock position to a post or to the bottom of the document
 */
function checkBottom() {
	const height = document.body.scrollHeight - window.innerHeight
	atBottom = height <= window.scrollY
	if (lockIndicator) {
		lockIndicator.style.visibility = atBottom ? 'visible' : 'hidden'
	}
}

checkBottom()
document.addEventListener('scroll', checkBottom)

// Cache previous reference element to minimize DOM lookup
let referenceEl

/**
 * Lock position to the bottom of a thread or keep the viewport from bumping
 * on out of sight DOM mutation.
 * @param {function} func
 * @returns {*}
 */
export function followDOM(func) {
	const previous = referenceDistance(),
		ret = func()

	// Prevent scrolling with new posts, if page isn't visible
	if (atBottom && (!document.hidden || options.get('alwaysLock'))) {
		window.scrollTo(0,  document.body.scrollHeight)
	} else {
		// Element was removed or something
		if (!elExists(referenceEl)) {
			return ret
		}

		// Only compensate, if the height increased above the viewport
		const delta = topDistance(referenceEl, true) - previous
		if (delta) {
			window.scrollBy(0, delta)
		}
	}
	return ret
}

/**
 * Check if element reference exists and is in the DOM
 * @param {Element} el
 */
function elExists(el) {
	return el && document.contains(el)
}

/**
 * Return element position dimentions against the viewport, if the element
 * is withing the viewport
 * @param {Element} el
 * @param {bool} skipCheck
 * @returns {(Number|null)}
 */
function topDistance(el, skipCheck) {
	const {top} = el.getBoundingClientRect()
	if (skipCheck || (top >= 0 && top < window.innerHeight)) {
		return top
	}
	return null
}

/**
 * Returns distance of viewport to with current reference element
 * @returns {(Number|null)}
 */
function referenceDistance() {
	if (elExists(referenceEl)) {
		const bounds = topDistance(referenceEl)
		if (bounds !== null) {
			return bounds
		}
	}

	// Find new reference element (first inside viewport). Account for empty
	// threads and boards with no artciles or sections.
	for (let selector of ['article', 'section', 'threads']) {
		for (let el of $threads.queryAll(selector)) {
			const bounds = topDistance(el)
			if (bounds !== null) {
				referenceEl = el
				return bounds
			}
		}
	}
}

/**
 * Account for banner height, when scrolling to an anchor
 */
function aboveBanner (){
	if (!/^#p\d+$/.test(location.hash)) {
		return
	}
	const anchor = document.query(location.hash)
	if (!anchor) {
		return
	}
	window.scrollTo(0, topDistance(anchor) - $banner.height)
}
events.reply('scroll:aboveBanner', aboveBanner)
window.onload = aboveBanner
