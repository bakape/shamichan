/*
Common utility functions
 */

const main = require('main');

// Gather salected post checkboxes
function getSelected() {
	const checked = [];
	loopCheckboxes(el => el.checked && checked.push(main.etc.getID(el)));

	// Postforms will not have an ID, so we remove falsy values
	return main._.compact(checked);
}
exports.getSelected = getSelected;

function loopCheckboxes(func) {
	const els = main.$threads[0].getElementsByClassName('postCheckbox');
	for (let i = 0; i < els.length; i++) {
		func(els[i]);
	}
}
exports.loopCheckboxes = loopCheckboxes;
