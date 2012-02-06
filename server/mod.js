function (AUTH) {

var $panel;

function show_panel() {
	if ($panel)
		return;
	var $del = $('<input type=button value=Delete>').click(korosu);
	$panel = $('<div></div>').append($del).css({
		position: 'fixed', bottom: 0, right: 0
	}).appendTo('body');
}

function korosu() {
	var ids = [];
	$('header>input').each(function () {
		var $check = $(this);
		if ($check.attr('checked')) {
			var id = $check.parent().parent().attr('id');
			ids.push(parseInt(id));
		}
	});
	if (ids.length) {
		ids.unshift(5);
		send(ids);
	}
	else {
		var $button = $(this);
		var caption = _.bind($button.val, $button);
		caption('Nothing selected.');
		_.delay(caption, 2000, 'Delete');
	}
}

readOnly.push('graveyard');

window.fun = function () {
	send([10, THREAD]);
};

override(PF, 'make_alloc_request', function (orig, text) {
	var msg = orig.call(this, text);
	if ($('#authname').attr('checked'))
		msg.auth = AUTH;
	return msg;
});

$(document).click(function (event) {
	var $box = $(event.target);
	if ($box.attr('type') == 'checkbox' && $box.parent('header').length)
		show_panel();
});

$(document).ready(function () {
	$('h1').text('Moderation - ' + $('h1').text());
	$('<input type=checkbox>').insertBefore('header>:first-child');
	$name.after(' <input type=checkbox id="authname">' +
			' <label for="authname">' + AUTH + '</label>');

	oneeSama.hook('afterInsert', function (target) {
		$('<input type=checkbox>').insertBefore(target.find(
				'>header>:first-child'));
	});
});

}
