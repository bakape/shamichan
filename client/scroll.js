/*
 * Various page scrolling logic
 */

var $ = require('jquery');

// Account for banner height, when scrolling to an anchor

function aboveBanner (){
	if (!/^#\d+$/.test(location.hash))
		return;
	$(window).scrollTop($(location.hash).offset().top - $('#banner').height());
}
exports.aboveBanner = aboveBanner;

window.onload = exports.aboveBanner;
