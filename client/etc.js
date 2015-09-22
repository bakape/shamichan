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

// Parse HTML string to node collection
function parseDOM(string) {
	const el = document.createElement('div');
	el.innerHTML = string;
	return el.childNodes;
}
exports.parseDOM = parseDOM;

// Add an event listener that filters targets according to a CSS selector
function listener(el, type, selector, handler) {
	el.addEventListener(type, function (event) {
		if (event.target.matches(selector))
			handler.call(this, event);
	});
}
exports.listener = listener;
