/*
 Name, email, tripcode and staff title persistence and postform propagation
 */

let main = require('../main'),
	{$, $script, $email, $name, _, common, config} = main;

function load() {
	try {
		const id = JSON.parse(localStorage.ident);
		if (id.name)
			$name.val(id.name);
		if (id.email)
			$email.val(id.email);
	}
	catch(e) {}
}

let save = _.debounce(function() {
	try {
		const name = $name.val();
		let email = $email.val();
		// Staff login method
		if (email === config.LOGIN_KEYWORD) {
			$email.val('');
			$script(config.MEDIA_URL + 'js/login.js?v=' + main.clientHash);
			email = false;
		}

		if (name || email) {
			let id = {};
			if (name)
				id.name = name;
			if (email)
				id.email = email;
			localStorage.ident = JSON.stringify(id);
		}
		else
			localStorage.removeItem('ident');
	}
	catch(e) {}
}, 1000);

// Sync persistance and postForm with input changes
function propagate() {
	let postForm = main.request('postForm');
	if (postForm)
		postForm.renderIdentity();
	save();
}

main.defer(function() {
	load();
	$name.on('input', propagate);
	$email.on('input', propagate);
});
