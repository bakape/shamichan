/*
 * Various page scrolling logic
 */

const main = require('./main'),
	{$, Backbone, options, state} = main;

const {body} = document;
let atBottom, lockIndicator, isThread;

function cacheState() {
	lockIndicator = document.query('#lock');
	isThread = !!state.posts.get('thread');
}

// Recache state on page change
state.page.on('change', cacheState);
cacheState();

// Sets the scroll lock position to a post or to bottom of the document
function checkBottom() {
	atBottom = (body.scrollHeight - window.innerHeight) <= window.scrollY;
	if (lockIndicator)
		lockIndicator.style.visibility = atBottom ? 'visible' : 'hidden';
}

checkBottom();
document.addEventListener('scroll', checkBottom);

/*
 Lock position to the bottom of a thread or keep the viewport from bumping
 on out of sight DOM mutation.
 */
// Cache previous reference element to minimize DOM lookup
let referenceEl;

function followDOM(func) {
	const previous = referenceDistance(),
		ret = func();

	// Prevent scrolling with new posts, if page isn't visible
	if (atBottom && (!document.hidden || options.get('alwaysLock')))
		window.scrollTo(0,  body.scrollHeight);
	else {
		// Element was removed or something
		if (!elExists(referenceEl))
			return ret;

		// Only compensate, if the height increased ~above the viewport
		const delta = topDistance(referenceEl, true) - previous;
		if (delta)
			window.scrollBy(0, delta);
	}
	return ret;
}
main.follow = followDOM;

// Check if element reference exists and is in the DOM
function elExists(el) {
	return el && document.contains(el);
}

// Return element position dimentions against the viewport, if the element
// is withing the viewport
function topDistance(el, skipCheck) {
	const {top} = el.getBoundingClientRect();
	if (skipCheck || (top >= 0 && top < window.innerHeight))
		return top;
	return null;
}

function referenceDistance() {
	if (elExists(referenceEl)) {
		const bounds = topDistance(referenceEl);
		if (bounds !== null)
			return bounds;
	}

	// Find new reference element (first inside viewport). Account for empty
	// threads and boards with no artciles or sections.
	for (let selector of ['article', 'section', 'threads']) {
		for (let el of main.$threads[0].queryAll(selector)) {
			const bounds = topDistance(el);
			if (bounds !== null) {
				referenceEl = el;
				return bounds;
			}
		}
	}
}

// Account for banner height, when scrolling to an anchor
function aboveBanner (){
	if (!/^#\d+$/.test(location.hash))
		return;
	let $anchor = $(location.hash);
	if (!$anchor.length)
		return;
	$(window).scrollTop($anchor.offset().top - $('#banner').height());
}
main.reply('scroll:aboveBanner', aboveBanner);
window.onload = aboveBanner;
