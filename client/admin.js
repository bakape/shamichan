/* NOTE: This file is processed by server/state.js
         and served by server/server.js (to auth'd users only) */

var $selectButton, $controls;
window.loggedInUser = IDENT.email;
window.x_csrf = IDENT.csrf;

function show_panel() {
	var specs = [
		{name: 'Lewd', kind: 7},
		{name: 'Porn', kind: 8},
		{name: 'Delete', kind: 9},
		{name: 'Lock', kind: 11},
	];
	var $panel = $('<div></div>', {
		css: {'margin': '0.5em 0.5em 0.5em 1em'},
	});

	$selectButton = $('<input />', {
		type: 'button', val: 'Select',
		click: function (e) { toggle_multi_selecting(); },
	});
	$panel.append($selectButton, ' ');

	$controls = $('<span></span>').hide();
	_.each(specs, function (spec) {
		$controls.append($('<input />', {
			type: 'button',
			val: spec.name,
			data: {kind: spec.kind},
		}), ' ');
	});
	$controls.on('click', 'input[type=button]', panel_action);

	_.each(delayNames, function (when, i) {
		var id = 'delay-' + when;
		var $label = $('<label/>', {text: when, 'for': id});
		var $radio = $('<input/>', {
			id: id, type: 'radio', val: when, name: 'delay',
		});
		if (i == 0)
			$radio.prop('checked', true);
		$controls.append($radio, $label, ' ');
	});

	$panel.append($controls).insertBefore(THREAD ? 'hr:last' : $ceiling);
}

function panel_action(event) {
	var ids = [];
	var $sel = $('.selected');
	$sel.each(function () {
		var id = extract_num(parent_post($(this)));
		if (id)
			ids.push(id);
	});
	var $button = $(this);
	var kind = $button.data('kind');

	/* On a thread page there's only one thread to lock, so... */
	if (kind == 11 && THREAD && !ids.length)
		ids = [THREAD];

	if (ids.length) {
		var when = $('input:radio[name=delay]:checked').val();
		ids.unshift(parseInt(kind, 10), {when: when});
		send(ids);
		$sel.removeClass('selected');
	}
	else {
		var orig = $button.val();
		var caption = _.bind($button.val, $button);
		caption('Nope.');
		if (orig != 'Nope.')
			_.delay(caption, 2000, orig);
	}
}

readOnly.push('graveyard');
menuOptions.unshift('Select');

var multiSelecting = false;

function toggle_multi_selecting($post) {
	var oldTarget;
	if ($post) {
		oldTarget = lockTarget;
		set_lock_target(extract_num($post));
	}
	with_dom(function () {
	multiSelecting = !multiSelecting;
	if (multiSelecting) {
		$('body').addClass('multi-select');
		make_selection_handle().prependTo('article');
		make_selection_handle().prependTo('section > header');
		if ($post)
			$post.find('.select-handle:first'
					).addClass('selected');
		$controls.show();
		$selectButton.val('X');
	}
	else {
		$('body').removeClass('multi-select');
		$('.select-handle').remove();
		$controls.hide();
		$selectButton.val('Select');
	}
	});
	if ($post)
		set_lock_target(oldTarget);
}

menuHandlers.Select = toggle_multi_selecting;

function make_selection_handle() {
	return $('<a class="select-handle" href="#"/>');
}

window.fun = function () {
	send([33, THREAD]);
};

override(ComposerView.prototype, 'make_alloc_request',
			function (orig, text, img) {
	var msg = orig.call(this, text, img);
	if ($('#authname').is(':checked'))
		msg.auth = IDENT.auth;
	return msg;
});

$DOC.on('click', '.select-handle', function (event) {
	event.preventDefault();
	$(event.target).toggleClass('selected');
});

$(function () {
	$('h1').text('Moderation - ' + $('h1').text());
	var $authname = $('<input>', {type: 'checkbox', id: 'authname'});
	var $label = $('<label/>', {text: ' '+IDENT.auth}).prepend($authname);
	$name.after(' ', $label);

	/* really ought to be done with model observation! */
	$authname.change(function () {
		if (postForm)
			postForm.propagate_ident();
	});
	oneeSama.hook('fillMyName', function ($el) {
		var auth = $('#authname').is(':checked');
		$el.toggleClass(IDENT.auth.toLowerCase(), auth);
		if (auth)
			$el.append(' ## ' + IDENT.auth)
	});

	oneeSama.hook('afterInsert', function (target) {
		if (multiSelecting)
			make_selection_handle().prependTo(target);
	});
	show_panel();
});
