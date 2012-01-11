(function () {

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
		ids.unshift(5, document.cookie);
		send(ids);
	}
	else {
		var $button = $(this);
		var caption = _.bind($button.val, $button);
		caption('Nothing selected.');
		_.delay(caption, 2000, 'Delete');
	}
}

if (BOARD == 'graveyard') {
	MIRU.connect = function () {
		sync_status('Syncing...', false);
		send([9, BOARD, syncs, BUMP, document.cookie]);
	};
	insert_pbs = function () {};
}

window.fun = function () {
	send([10, document.cookie, THREAD]);
};

function make_alloc_admin(text) {
	var msg = this.make_alloc_vanilla(text);
	if ($('#admin').attr('checked'))
		msg.auth = 'Admin';
	if (msg.auth)
		msg.cookie = document.cookie;
	return msg;
}

$(document).click(function (event) {
	var $box = $(event.target);
	if ($box.attr('type') == 'checkbox' && $box.parent('header').length)
		show_panel();
});

$(document).ready(function () {
	$('h1').text('Moderation - ' + $('h1').text());
	$('<input type=checkbox>').insertBefore('header>:first-child');
	$name.after(' <input type=checkbox id=admin>' +
			'<label for=admin>Admin</label>');

	/* Dumb hack, injecting auth. Should inherit or something? */
	var pfp = PostForm.prototype;
	pfp.make_alloc_vanilla = pfp.make_alloc_request;
	pfp.make_alloc_request = make_alloc_admin;

	oneeSama.check = function (target) {
		$('<input type=checkbox>').insertBefore(target.find(
				'>header>:first-child'));
	};
});

})();
