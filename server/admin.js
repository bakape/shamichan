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
	$('article header input').each(function () {
		var $check = $(this);
		if ($check.attr('checked')) {
			var id = $check.parents('article').attr('id');
			ids.push(parseInt(id));
		}
	});
	if (ids.length) {
		ids.unshift(5);
		send(ids);
	}
}

$(document).click(function (event) {
	var $box = $(event.target);
	if ($box.attr('type') == 'checkbox' && $box.parent('header').length)
		show_panel();
});

$(document).ready(function () {
	$('h1').text('Moderation - ' + $('h1').text());
	$('<input type=checkbox>').insertBefore('article>header>b');
});

})();
