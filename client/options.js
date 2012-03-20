var $name, $email;
var options, toggles = {}, $opts;
var inputMinSize = 300, nashi;

(function () {
	try {
		options = JSON.parse(localStorage.options);
	}
	catch (e) { }
	if (!options)
		options = {};

	nashi = {
		opts: [],
		upload: !!$('<input type="file"/>').prop('disabled'),
	};
	if (window.screen && screen.width <= 320)
		inputMinSize = 50;
	if ('ontouchstart' in window)
		nashi.opts.push('preview');
})();


function load_ident() {
	try {
		var id;
		// TEMP migration
		var oldName = localStorage.getItem('name');
		var oldEmail = localStorage.getItem('email');
		if (oldName || oldEmail) {
			id = {};
			if (oldName)
				id.name = oldName;
			if (oldEmail)
				id.email = oldEmail;
			localStorage.setItem('ident', JSON.stringify(id));
		}
		else {
			id = JSON.parse(localStorage.getItem('ident'));
		}
		localStorage.removeItem('name');
		localStorage.removeItem('email');

		if (id.name)
			$name.val(id.name);
		if (id.email)
			$email.val(id.email);
	}
	catch (e) {}
}

function save_ident() {
	try {
		var name = $name.val(), email = $email.val();
		if (is_sage(email) && !is_noko(email))
			email = false;
		var id = {};
		if (name || email) {
			if (name)
				id.name = name;
			if (email)
				id.email = email;
			localStorage.setItem('ident', JSON.stringify(id));
		}
		else
			localStorage.removeItem('ident');
	}
	catch (e) {}
}

/* Pre-load options window */
function opt_change(id, b) {
	return function (event) {
		options[id] = $(this).prop('checked');
		try {
			localStorage.options = JSON.stringify(options);
		}
		catch (e) {}
		b(options[id]);
	};
}

function toggle_opts() {
	$opts.toggle('fast');
}

$(function () {
	$name = $('input[name=name]');
	$email = $('input[name=email]');
	load_ident();
	var save = _.debounce(save_ident, 1000);
	function prop() {
		if (postForm)
			postForm.propagate_ident();
		save();
	}
	$name.input(prop);
	$email.input(prop);

  	$opts = $('<div class="modal"/>');
	for (var id in toggles) {
		if (nashi.opts.indexOf(id) >= 0)
			continue;
		var b = toggles[id];
		$('<input type="checkbox" id="'+id+'" /> <label for="' +
				id + '">' + b.label + '</label><br>'
			).appendTo($opts).change(opt_change(id, b)
			).prop('checked', options[id] ? 'checked' : null);
		b(options[id]);
	}
	$opts.hide().appendTo(document.body);
	$('<a id="options">Options</a>').click(toggle_opts
			).insertAfter('#sync');
	toggles = null;
});
