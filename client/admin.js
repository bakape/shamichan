var $panel;

function show_panel() {
	var specs = [
		{name: 'Select', kind: 'select'},
		{name: 'Spoil', kind: 7},
		{name: 'Lewd', kind: 8},
		{name: 'Delete', kind: 9},
		{name: 'Lock', kind: 11},
	];
	$panel = $('<div></div>', {css: {'margin': '0.5em 0.5em 0.5em 1em'}});
	_.each(specs, function (spec) {
		$panel.append($('<input />', {
			type: 'button',
			val: spec.name,
			data: {kind: spec.kind},
		}), ' ');
	});
	$panel.on('click', 'input', panel_click).insertBefore(
			THREAD ? 'hr:last' : $ceiling);
}

function panel_click(event) {
	var ids = [];
	var $sel = $('.selected');
	$sel.each(function () {
		var id = extract_num(parent_post($(this)));
		if (id)
			ids.push(id);
	});
	var $button = $(this);
	var kind = $button.data('kind');
	if (kind == 'select')
		toggle_multi_selecting(null);
	else if (ids.length) {
		ids.unshift(parseInt(kind, 10));
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
	if (!multiSelecting) {
		$('body').addClass('multi-select');
		make_selection_handle().prependTo('article');
		make_selection_handle().prependTo('section > header');
		if ($post)
			$post.find('.select-handle:first'
					).addClass('selected');
		multiSelecting = true;
	}
	else {
		$('body').removeClass('multi-select');
		$('.select-handle').remove();
		multiSelecting = false;
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

override(PF, 'make_alloc_request', function (orig, text, img) {
	var msg = orig.call(this, text, img);
	if ($('#authname').attr('checked'))
		msg.auth = AUTH;
	return msg;
});

$DOC.on('click', '.select-handle', function (event) {
	event.preventDefault();
	$(event.target).toggleClass('selected');
});

$(function () {
	$('h1').text('Moderation - ' + $('h1').text());
	$name.after(' <input type=checkbox id="authname">' +
			' <label for="authname">' + AUTH + '</label>');
	$email.after(' <form action="../logout" method=POST ' +
			'style="display: inline">' +
			'<input type=submit value=Logout></form>');

	oneeSama.hook('afterInsert', function (target) {
		if (multiSelecting)
			make_selection_handle().prependTo(target);
	});
	show_panel();
});
