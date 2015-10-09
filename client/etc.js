/*
 Client-side helper functions
 */

const main = require('./main'),
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

function getNum(el) {
	return el && parseInt(el.getAttribute('id').slice(1), 10);
}
exports.getNum = getNum;

function getID(el) {
	return el && getNum(el.closest('article, section'));
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
function parseDOM(string, forceArray) {
	const el = document.createElement('div');
	el.innerHTML = string;
	const children = el.childNodes;
	if (!forceArray && children.length === 1)
		return children[0];
	return Array.from(children);
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

function once(el, type, handler) {
	el.addEventListener(type, function (event) {
		handler.call(this, event);
		el.removeEventListener(type, handler);
	});
}
exports.once = once;

// Width of element padding and margin
function outerWidth(el) {
	const style =  getComputedStyle(el);
	let width = 0;
	for (let prop of ['marginLeft', 'marginRight', 'paddingLeft',
		'paddingRight']
	) {
		width += parseInt(style[prop]);
	}
	return width;
}
exports.outerWidth = outerWidth;
