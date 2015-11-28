/*
Toggle and preview staff titles on new posts
 */

const main = require('main'),
	{lang} = main,
	{auth} = main.ident;

// Insert toggler checkbox into name form
main.$name.after(main.common.parseHTML
	`<label title="${lang.mod.title[1]}" class="mod">
		<input type="checkbox" id="authName">
		 ${lang.mod.title[0]}
	 </label>`);

const $authName = main.$('#authName');

// Preview the title in postForm
main.oneeSama.hook('fillMyName', $el => {
	const checked = $authName[0].checked;
	$el.toggleClass(auth === 'admin' ? 'admin' : 'moderator', checked);
	if (checked) {
		$el.append(' ## ' + main.state.hotConfig
			.get('staff_aliases')[auth] || auth);
	}
});
$authName.change(() => main.request('postForm:indentity'));

// Extend default allocation request
override(main.posts.posting.ComposerView.prototype, 'allocationMessage',
	function (orig, ...args) {
		const msg = orig.call(this, ...args);
		if ($authName[0].checked)
			msg.auth = auth;
		return msg;
	});

// Override a method on an object with a new method
function override(parent, method, upgrade) {
	const orig = parent[method];
	parent[method] = function (...args) {
		return upgrade.call(this, orig, ...args);
	}
}
