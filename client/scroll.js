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
 Logic for locking position to the bottom of a thread or keeping the
  viewport scroll position unchanged.
 */
function followDOM(func) {
	const previous = body.scrollHeight,
		ret = func.call(this),
		delta = body.scrollHeight - previous;
	if (delta) {
		if (!atBottom)
			window.scrollBy(0, delta);
		// Prevent scrolling with new posts, if page isn't visible
		else if (!document.hidden || options.get('alwaysLock'))
			window.scrollTo(0,  body.scrollHeight);
	}
	return ret;
}
// Shorthand; we use this a lot
main.follow = followDOM;

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
