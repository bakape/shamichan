/*
 Hide posts you don't like
 */

let main = require('./main');

// Remember hidden posts for 7 days only, to perevent the cookie from
// eclipsing the Sun
let hidden = new main.memory('hide', 7);

main.comply('hide', function(model) {
	// Hiding your own posts would open up the gates for a ton of bugs. Fuck
	// that.
	if (model.get('mine'))
		return;
	const count = hidden.write(model.get('num'), hidden.now());
	model.remove();
	// Forward number to options menu
	main.command('hide:render', count);
});

// Clear hidden
main.comply('hide:clear', () => hidden.purge_all());

hidden.purge_expired_soon();

// Initial render
main.defer(() => main.command('hide:render', hidden.size()));
