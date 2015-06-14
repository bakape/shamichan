/*
 Client-side helper functions
 */

let main = require('./main'),
	{_, state} = main;

// For mobile
function touchable_spoiler_tag(del) {
	del.html = '<del onclick="void(0)">';
}
exports.touchable_spoiler_tag = touchable_spoiler_tag;
main.oneeSama.hook('spoilerTag', touchable_spoiler_tag);

function imageUploadURL() {
	return (main.config.UPLOAD_URL || '../upload/') + '?id='
		+ state.page.get('connID');
}
exports.uploadURL = imageUploadURL;

// Keep the UI from locking as the loop iterates
function defferLoop(items, func, i) {
	i || (i = 0);
	func(items[i]);
	if (++i < items.length)
		_.defer(defferLoop, items, func, i);
}
exports.defferLoop = defferLoop;
