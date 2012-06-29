var nashi = {opts: []}, inputMinSize = 300;

function extract_num(q) {
	return parseInt(q.attr('id'), 10);
}

function parent_post($el) {
	var $post = $el.parents('article');
	return $post.length ? $post : $el.parents('section');
}

(function () {

nashi.upload = !!$('<input type="file"/>').prop('disabled');

if (window.screen && screen.width <= 320)
	inputMinSize = 50;
if ('ontouchstart' in window)
	nashi.opts.push('preview');

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

var optSpecs = [];
function add_spec(id, label, func, type) {
	id = id.replace(/\$BOARD/g, BOARD);
	if (!func)
		func = function () {};
	optSpecs.unshift({id: id, label: label, func: func, type: type});
}

/* THEMES */

var themes = ['moe', 'gar', 'mawaru', 'moon'];
var globalVersion = 5;

add_spec('board.$BOARD.theme', 'Theme', function (theme) {
	if (theme) {
		var css = theme + '-v' + globalVersion + '.css';
		$('#theme').attr('href', mediaURL + css);
	}
}, themes);

/* BACKLINKS */

add_spec('backlinks', 'Local backlinks', function (b) {
	if (b)
		show_backlinks();
	else
		$('small').remove();
}, 'checkbox');

function show_backlinks() {
	if (!CurThread)
		return;
	if (load_page_backlinks) {
		load_page_backlinks();
		load_page_backlinks = null;
	}
	else {
		CurThread.each(function (reply) {
			if (reply.has('backlinks'))
				reply.trigger('change:backlinks');
		});
	}
}

var load_page_backlinks = with_dom(function () {
	$('blockquote a').each(function () {
		var $a = $(this);
		var m = $a.attr('href').match(/^#(\d+)$/);
		if (!m)
			return;
		var destId = parseInt(m[1], 10);
		if (!CurThread.get(destId)) // local backlinks only for now
			return;
		var src = CurThread.get(extract_num(parent_post($a)));
		if (!src)
			return;
		var update = {};
		update[destId] = THREAD;
		add_post_links(src, update);
	});
});

/* HOVER PREVIEW */

var preview, previewNum;

add_spec('preview', 'Hover preview', function (b) {
	if (b)
		$(document).mousemove(hover_shita);
	else
		$(document).unbind('mousemove', hover_shita);
}, 'checkbox');

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

function preview_miru(event, num) {
	if (num != previewNum) {
		var post = $('#' + num);
		if (!post.length)
			return false;
		if (preview)
			preview.remove();
		var bits = post.children();

		/* stupid hack, should be using views */
		if (bits[0] && $(bits[0]).is('.select-handle'))
			bits = bits.slice(1);

		if (post.is('section'))
			bits = bits.slice(0, 3);
		preview = $('<div class="preview"/>').append(bits.clone());
	}
	var width = preview.width();
	var height = preview.height();
	if (height < 5) {
		preview.hide();
		$(document.body).append(preview);
		width = preview.width();
		height = preview.height();
		preview.detach().show();
	}
	var x = event.pageX + 20;
	var y = event.pageY - height - 20;
	if (x + width > window.innerWidth)
		x = Math.max(0, event.pageX - width - 20);
	preview.css({left: x + 'px', top: y + 'px'});
	if (num != previewNum) {
		$(document.body).append(preview);
		previewNum = num;
	}
	return true;
}

/* INLINE EXPANSION */

if (window.devicePixelRatio > 1)
	add_spec('nohighres', 'High-res expansions', null, 'revcheckbox');

add_spec('inline', 'Inline image expansion', null, 'checkbox');

$(document).on('mouseup', function (event) {
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
});

$(document).on('click', 'img', function (event) {
	if (options.inline) {
		var $target = $(event.target);
		if (!$target.data('skipExpand'))
			toggle_expansion($target, event);
	}
});

function toggle_expansion(img, event) {
	var href = img.parent().attr('href');
	if (href.match(/^\.\.\/outbound\//))
		return;
	event.preventDefault();
	var expand = !img.data('thumbSrc');
	var $imgs = img;
	if (THREAD && (event.altKey || event.shiftKey)) {
		var post = img.closest('article');
		if (post.length)
			$imgs = post.nextAll(':has(img):lt(4)').andSelf();
		else
			$imgs = img.closest('section').children(
					':has(img):lt(5)');
		$imgs = $imgs.find('img');
	}

	with_dom(function () {
		$imgs.each(function () {
			var $img = $(this);
			if (expand)
				expand_image($img);
			else {
				contract_image($img, event);
				event = null; // de-zoom to first image only
			}
		});
	});
}

function contract_image($img, event) {
	var thumb = $img.data('thumbSrc');
	if (!thumb)
		return;
	// try to keep the thumbnail in-window for large images
	var h = $img.height();
	var th = parseInt($img.data('thumbHeight'), 10);
	if (event) {
		var y = $img.offset().top, t = $(window).scrollTop();
		if (y < t && th < h)
			window.scrollBy(0, Math.max(th - h,
					y - t - event.clientY + th/2));
	}
	$img.replaceWith($('<img>')
			.width($img.data('thumbWidth')).height(th)
			.attr('src', thumb));
}

function expand_image($img) {
	if ($img.data('thumbSrc'))
		return;
	var a = $img.parent();
	var href = a.attr('href');
	if (!href)
		return;
	var dims = a.prev().text().match(/(\d+)x(\d+)/);
	if (!dims)
		return;
	var w = parseInt(dims[1], 10), h = parseInt(dims[2], 10);
	var r = window.devicePixelRatio;
	if (!options.nohighres && r && r > 1) {
		w /= r;
		h /= r;
	}
	$img.replaceWith($('<img>').data({
		thumbWidth: $img.width(),
		thumbHeight: $img.height(),
		thumbSrc: $img.attr('src'),
	}).attr('src', href).width(w).height(h));
}

(function () {
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
		var $o = $(event.target), id = $o.attr('id'), val;
		var spec = _.find(optSpecs, function (s) {
			return s.id == id;
		});
		if (spec.type == 'checkbox')
			val = !!$o.prop('checked');
		else if (spec.type == 'revcheckbox')
			val = !$o.prop('checked');
		else
			val = $o.val();
		options[id] = val;
		save_opts();
		(spec.func)(val);
	});
	_.each(optSpecs, function (spec) {
		var id = spec.id;
		if (nashi.opts.indexOf(id) >= 0)
			return;
		var val = options[id], $input, type = spec.type;
		if (type == 'checkbox' || type == 'revcheckbox') {
			var b = (type == 'revcheckbox') ? !val : val;
			$input = $('<input type="checkbox" />')
				.prop('checked', b ? 'checked' : null);
		}
		else if (type instanceof Array) {
			$input = $('<select/>');
			_.each(type, function (item) {
				$('<option/>')
					.text(item).val(item)
					.appendTo($input);
			});
			if (type.indexOf(val) >= 0)
				$input.val(val);
		}
		var $label = $('<label/>').attr('for', id).text(spec.label);
		$opts.append($input.attr('id', id), ' ', $label, '<br>');
		(spec.func)(val);
	});
	$opts.hide().appendTo(document.body);
	$('<a id="options">Options</a>').click(function () {
		$opts.toggle('fast');
	}).insertAfter('#sync');
})();

})();
