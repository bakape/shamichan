/*
 * Various page scrolling logic
 */

var $ = require('jquery');

// Account for banner height, when scrolling to an anchor
exports.aboveBanner = function (){
	if (!/^#\d+$/.test(location.hash))
		return;
	$(window).scrollTop($(location.hash).offset().top - $('#banner').height());
};

window.onload = exports.aboveBanner;