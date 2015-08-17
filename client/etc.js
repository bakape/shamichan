/*
 Client-side helper functions
 */

let main = require('./main'),
	{$, _, state} = main;

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

function getID(el) {
	return parseInt($(el).closest('article, section').attr('id'), 10);
}
exports.getID = getID;

function getModel(el) {
	const id = getID(el);
	if (!id)
		return null;
	return state.posts.get(id);
}
exports.getModel = getModel;
