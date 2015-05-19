/*
 Client-side helper functions
 */

let main = require('./main');

// For mobile
function touchable_spoiler_tag(del) {
	del.html = '<del onclick="void(0)">';
}
exports.touchable_spoiler_tag = touchable_spoiler_tag;
main.oneeSama.hook('spoilerTag', touchable_spoiler_tag);
