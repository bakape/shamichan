var $name, $email;
var options, toggles = {};
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
		var id = JSON.parse(localStorage.ident);
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

function save_opts() {
	try {
		localStorage.options = JSON.stringify(options);
	}
	catch (e) {}
}

function hover_shita(event) {
	if (event.target.tagName.match(/^A$/i)) {
		var m = $(event.target).text().match(/^>>(\d+)$/);
		if (m && preview_miru(event, parseInt(m[1], 10)))
			return;
	}
	if (preview) {
		preview.remove();
		preview = previewNum = null;
	}
}

function mouseup_shita(event) {
	/* Bypass expansion for non-left mouse clicks */
	if (options.inline && event.which > 1) {
		var img = $(event.target);
		if (img.is('img')) {
			img.data('skipExpand', true);
			setTimeout(function () {
				img.removeData('skipExpand');
			}, 100);
		}
	}
}

toggles.inline = function (b) {
	if (b)
		$(document).mouseup(mouseup_shita);
	else
		$(document).unbind('mouseup', mouseup_shita);
};
toggles.inline.label = 'Inline image expansion';
toggles.preview = function (b) {
	if (b)
		$(document).mousemove(hover_shita);
	else
		$(document).unbind('mousemove', hover_shita);
};
toggles.preview.label = 'Hover preview';

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

	var $opts = $('<div class="modal"/>').change(function (event) {
		var $o = $(event.target), id = $o.attr('id');
		var val = options[id] = !!$o.prop('checked');
		save_opts();
		toggles[id](val);
	});
	for (var id in toggles) {
		if (nashi.opts.indexOf(id) >= 0)
			continue;
		var val = options[id], b = toggles[id];
		var $check = $('<input type="checkbox" />')
			.attr('id', id)
			.prop('checked', val ? 'checked' : null);
		var $label = $('<label/>').attr('for', id).text(b.label);
		$opts.append($check, ' ', $label, '<br>');
		b(val);
	}
	$opts.hide().appendTo(document.body);
	$('<a id="options">Options</a>').click(function () {
		$opts.toggle('fast');
	}).insertAfter('#sync');
});
