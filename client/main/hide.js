/*
 Hide posts you don't like
 */

let main = require('./main');

// Remember hidden posts for 7 days only to prevent the cookie from
// eclipsing the Sun
let hidden = new main.Memory('hide', 7, true);

main.reply('hide', function(model) {
	// Hiding your own posts would open up the gates for a ton of bugs. Fuck
	// that.
	if (model.get('mine'))
		return;
	const count = hidden.write(model.get('num'));
	model.remove();

	// Forward number to options menu
	main.request('hide:render', count);
});

main.reply('hide:clear', () => hidden.purgeAll());

// Initial render
main.defer(() => main.request('hide:render', hidden.size()));
