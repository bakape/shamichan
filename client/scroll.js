/*
 * Various page scrolling logic
 */

var $ = require('jquery');

// Account for banner height, when scrolling to an anchor

function aboveBanner (){
	if (!/^#\d+$/.test(location.hash))
		return;
	let $anchor = $(location.hash);
	if (!$anchor.length)
		return;
	$(window).scrollTop($anchor.offset().top - $('#banner').height());
}
exports.aboveBanner = aboveBanner;

window.onload = exports.aboveBanner;
