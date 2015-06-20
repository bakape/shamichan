/*
 Hide posts you don't like
 */

let main = require('./main');

// Remember hidden posts for 7 days only, to perevent the cookie from
// eclipsing the Sun
let hidden = new main.Memory('hide', 7, true);

main.comply('hide', function(model) {
	// Hiding your own posts would open up the gates for a ton of bugs. Fuck
	// that.
	if (model.get('mine'))
		return;
	const count = hidden.write(model.get('num'));
	model.remove();
	// Forward number to options menu
	main.command('hide:render', count);
});

main.comply('hide:clear', hidden.purgeAll);

// Initial render
main.defer(() => main.command('hide:render', hidden.size()));
