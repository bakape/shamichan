var $panel;
var nopeMsg = 'Nothing selected.';

function show_panel() {
	if ($panel)
		return;
	var specs = [
		{name: 'Spoiler', kind: 7},
		{name: 'Delete Image', kind: 8},
		{name: 'Delete', kind: 9},
	];
	$panel = $('<div></div>').css({
		position: 'fixed', bottom: '1em', right: '1em',
		"text-align": 'right'
	});
	var first = true;
	_.each(specs, function (spec) {
		if (!first)
			$panel.append('<br>');
		first = false;
		$('<input type=button>').val(spec.name).data('kind', spec.kind
				).click(korosu).appendTo($panel);
	});
	$panel.appendTo('body');
}

function korosu() {
	var ids = [];
	$('.selected').each(function () {
		var id = extract_num(parent_post($(this)));
		if (id)
			ids.push(id);
	});
	var $button = $(this);
	if (ids.length) {
		ids.unshift(parseInt($button.data('kind'), 10));
		send(ids);
		$('.selected').removeClass('selected');
	}
	else {
		var orig = $button.val();
		var caption = _.bind($button.val, $button);
		caption(nopeMsg);
		if (orig != nopeMsg)
			_.delay(caption, 2000, orig);
	}
}

readOnly.push('graveyard');
menuOptions.unshift('Select');

var multiSelecting = false;

menuHandlers['Select'] = function ($post) {
	var oldTarget = lockTarget;
	set_lock_target(extract_num($post));
	with_dom(function () {
	if (!multiSelecting) {
		$('body').addClass('multi-select');
		make_selection_handle().prependTo('article');
		make_selection_handle().prependTo('section > header');
		$post.find('.select-handle').addClass('selected');
		show_panel();
		multiSelecting = true;
	}
	else {
		$('body').removeClass('multi-select');
		$('.select-handle').remove();
		multiSelecting = false;
	}
	});
	set_lock_target(oldTarget);
};

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
});
