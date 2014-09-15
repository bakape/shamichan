var optSpecs = [];
var nashi = {opts: []}, inputMinSize = 300, fullWidthExpansion = false;
var shortcutKeys = {};

function extract_num(q) {
	return parseInt(q.attr('id'), 10);
}

function parent_post($el) {
	return $el.closest('article, section');
}

function parent_model($el) {
	var $a = parent_post($el);
	var op = extract_num($a);
	if (!op)
		return null;
	if ($a.is('section'))
		return Threads.get(op);
	var $s = $a.parent('section');
	if (!$s.length) {
		// when we have better hover/inline expansion we will have to
		// deal with this, probably by setting data-op on the post
		console.warn($a, "'s parent is not thread?!");
		return null;
	}
	var num = op;
	op = extract_num($s);
	return Threads.lookup(num, op);
}

(function () {

/* OPTIONS LIST */
optSpecs.push(option_inline_expansion);
if (window.devicePixelRatio > 1)
	optSpecs.push(option_high_res);
optSpecs.push(option_thumbs);
optSpecs.push(option_backlinks);
optSpecs.push(option_spoiler);
optSpecs.push(option_illya_dance);
optSpecs.push(option_illya_mute);
optSpecs.push(option_horizontal);
optSpecs.push(option_reply_at_right);
optSpecs.push(option_theme);
optSpecs.push(option_user_bg);
optSpecs.push(option_user_bg_set);
optSpecs.push(option_last_n);


_.defaults(options, {
	lastn: config.THREAD_LAST_N,
	inlinefit: 'width',
});
options = new Backbone.Model(options);


nashi.upload = !!$('<input type="file"/>').prop('disabled');

if (window.screen && screen.width <= 320) {
	inputMinSize = 50;
	fullWidthExpansion = true;
}

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
		if (email == 'misaki') {
			$email.val('');
			yepnope(mediaURL + 'js/login-v2.js');
			email = false;
		}
		else if (is_sage(email) && !is_noko(email))
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

options.on('change', function () {
	try {
		localStorage.options = JSON.stringify(options);
	}
	catch (e) {}
});

/* LAST N CONFIG */

function option_last_n(n) {
	if (!reasonable_last_n(n))
		return;
	$.cookie('lastn', n);
	// should really load/hide posts as appropriate
}
option_last_n.id = 'lastn';
option_last_n.label = '[Last #]';
option_last_n.type = 'positive';

oneeSama.lastN = options.get('lastn');
options.on('change:lastn', function (model, lastN) {
	oneeSama.lastN = lastN;
});

/* THEMES */

var themes = [
	'moe',
	'gar',
	'mawaru',
	'moon',
	'ashita',
	'console',
	'tea',
	'higan',
    'rave',
    'tavern',
    'glass'
];
var globalVersion = 8;

function option_theme(theme) {
	if (theme) {
		var css = theme + '-v' + globalVersion + '.css';
		$('#theme').attr('href', mediaURL + 'css/' + css);
	}
	gen_glass();
}

function gen_glass(){
	// Check if theme is glass and user-bg is set
	if (/glass/.test($('#theme').attr('href')) && $.cookie('user_bg') != '' && $.cookie('user_bg_state') == 'true'){
		var img = new Image();
		img.src = $.cookie('user_bg');
		img.onload = function(){
			$(this).remove(); // prevent memory leaks
			// Blur image with Pixastic and apply new backgrounds
			Pixastic.process(img, 'blurfast', {amount: 1.5}, function(blurred){
				var bg = 'url(' + blurred.toDataURL() + ') center fixed; background-size: cover;}' ;
				var gradient_dark = 'linear-gradient(rgba(40, 42, 46, 0.5), rgba(40, 42, 46, 0.5)),';
				var gradient_light = 'linear-gradient(rgba(145, 145, 145, 0.5), rgba(145, 145, 145, 0.5)),';
				$('body').append($('<style />', {
					id: 'blurred'
				}));
				$('#blurred').append(
					'article, aside, .pagination, .popup-menu, .modal, #FAQ, .preview, #banner, #banner_info {' +
						'background:' + gradient_dark + bg
				);
				$('#blurred').append('article.editing{' +
					'background:' + gradient_light + bg
				);
			});
		};
	} else
		$('#blurred').remove();
}

option_theme.id = 'board.$BOARD.theme';
option_theme.label = 'Theme';
option_theme.type = themes;

/* THUMBNAIL OPTIONS */

var revealSetup = false;

function option_thumbs(type) {
	$.cookie('thumb', type);
	// really ought to apply the style immediately
	// need pinky/mid distinction in the model to do properly
	oneeSama.thumbStyle = type;
	var hide = type == 'hide';
	if (hide)
		$('img').hide();
	else
		$('img').show();

	if (hide && !revealSetup)
		$DOC.on('click', 'article', reveal_thumbnail);
	else if (!hide && revealSetup)
		$DOC.off('click', 'article', reveal_thumbnail);
	revealSetup = hide;
}
option_thumbs.id = 'board.$BOARD.thumbs';
option_thumbs.label = 'Thumbnails';
option_thumbs.type = thumbStyles;

/* Alt-click a post to reveal its thumbnail if hidden */
function reveal_thumbnail(event) {
	if (!event.altKey)
		return;
	var $article = $(event.target);
	var $img = $article.find('img');
	if ($img.length) {
		with_dom(function () { $img.show(); });
		return false;
	}

	/* look up the image info and make the thumbnail */
	var thread = Threads.get(extract_num($article.closest('section')));
	if (!thread)
		return;
	var post = thread.get('replies').get(extract_num($article));
	if (!post)
		return;
	var info = post.get('image');
	if (!info)
		return;

	with_dom(function () {
		var img = oneeSama.gazou_img(info, false);
		var $img = $.parseHTML(flatten(img.html).join(''));
		$article.find('figcaption').after($img);
	});
	return false;
}

/* REPLY AT RIGHT */

function option_reply_at_right(r) {
	if (r)
		$('<style/>', {
			id: 'reply-at-right',
			text: 'aside { margin: -26px 0 2px auto; }',
		}).appendTo('head');
	else
		$('#reply-at-right').remove();
}
option_reply_at_right.id = 'replyright';
option_reply_at_right.label = '[Reply] at right';
option_reply_at_right.type = 'checkbox';

/* BACKLINKS */

function option_backlinks(b) {
	if (b)
		$('small').remove();
	else
		show_backlinks();
}
option_backlinks.id = 'nobacklinks';
option_backlinks.label = 'Backlinks';
option_backlinks.type = 'revcheckbox';

function show_backlinks() {
	if (load_thread_backlinks) {
		with_dom(function () {
			$('section').each(function () {
				load_thread_backlinks($(this));
			});
		});
		load_thread_backlinks = null;
		return;
	}

	Threads.each(function (thread) {
		thread.get('replies').each(function (reply) {
			if (reply.has('backlinks'))
				reply.trigger('change:backlinks');
		});
	});
}

var load_thread_backlinks = function ($section) {
	var op = extract_num($section);
	var replies = Threads.get(op).get('replies');
	$section.find('blockquote a').each(function () {
		var $a = $(this);
		var m = $a.attr('href').match(/^\d*#(\d+)$/);
		if (!m)
			return;
		var destId = parseInt(m[1], 10);
		if (!replies.get(destId)) // local backlinks only for now
			return;
		var src = replies.get(extract_num(parent_post($a)));
		if (!src)
			return;
		var update = {};
		update[destId] = op;
		add_post_links(src, update, op);
	});
};

/* SPOILER TOGGLE */


function option_spoiler(spoilertoggle) {
	$.cookie('spoil',spoilertoggle);
	oneeSama.spoilToggle = spoilertoggle;
}
option_spoiler.id = 'nospoilertoggle';
option_spoiler.label = 'Spoilered Images';
option_spoiler.type = 'revcheckbox';

/* ILLYA DANCE */

function option_illya_dance(illyatoggle){
	var muted = ' ';
	if ($.cookie('bgvid_mute') == 'true')
		muted = 'muted';
	var dancer = '<video autoplay ' + muted + ' loop id="bgvid" >' +
			'<source src="http://meguca.org/static/illya.webm" type="video/webm">' +
			'<source src="http://meguca.org/static/illya.mp4" type="video/mp4">' +
		'</video>';
	if (illyatoggle){
		$("body").append(dancer);
		$.cookie('bgvid', 'true');
	} else {
		$("#bgvid").remove();
		$.cookie('bgvid', 'false');
	}
}

option_illya_dance.id = 'board.$BOARD.illyaBGToggle';
option_illya_dance.label = 'Illya Dance';
option_illya_dance.type = 'checkbox';

function option_illya_mute(toggle){
	if (toggle)
		$.cookie('bgvid_mute', 'true');
	else 
		$.cookie('bgvid_mute', 'false');
		
	if ($.cookie('bgvid') == 'true'){
		option_illya_dance(false);
		option_illya_dance(true);
	}	
}

option_illya_mute.id = 'illyaMuteToggle';
option_illya_mute.label = 'Mute Illya';
option_illya_mute.type = 'checkbox';

/* HORIZONTAL POSTING */

function option_horizontal(toggle){
	var style = '<style id="horizontal">article,aside{display:inline-block;}</style>';
	if (toggle)
		$('body').append(style);
	else 
		$('#horizontal').remove();
}

option_horizontal.id = 'horizontalPosting';
option_horizontal.label = 'Horizontal Posting';
option_horizontal.type = 'checkbox';

/* CUSTOM USER-SET BACKGROUND */

function option_user_bg(toggle){
	if ($.cookie('user_bg') != '' && toggle){
		$.cookie('user_bg_state', 'true');
		var image = $.cookie('user_bg');		
		$('body').append($('<img />', {
			id: 'user_bg',
			src: image
		}));
		
		// Generate transparent BG, if theme is glass
		if (!$('#blurred').length)
			gen_glass();
		
		// Workaround for image unloading on tab focus loss Chrome bug
		// Basically, reloads the element to prevent the aggresive buggy caching or something
		if (typeof document.webkitHidden !== "undefined"){
			var hidden = "webkitHidden";
			var visibilityChange = "webkitvisibilitychange";
			
			function handleVisibilityChange(){
				if (document[hidden])
					$('#user_bg').attr('src', '');
				else
					$('#user_bg').attr('src', image);
			}
			document.addEventListener(visibilityChange, handleVisibilityChange, false);
		}	
	} else {
		$('#user_bg').remove();
		$('#blurred').remove();
		
		// Remove workaround listener
		if (typeof document.webkitHidden !== "undefined")
			document.removeEventListener(visibilityChange, handleVisibilityChange);
	}
}

option_user_bg.id = 'board.$BOARD.userBG';
option_user_bg.label = 'Custom Background';
option_user_bg.type = 'checkbox';

function option_user_bg_set(image){
	$.cookie('user_bg', image, {path: '/' + BOARD});
}

option_user_bg_set.id = 'board.$BOARD.userBGimage';
option_user_bg_set.label = ' ';
option_user_bg_set.type = 'image';

/* INLINE EXPANSION */

function option_inline_expansion() {
	/* TODO: do it live */
}
option_inline_expansion.id = 'inlinefit';
option_inline_expansion.label = 'Expansion';
option_inline_expansion.type = ['none', 'full', 'width', 'height', 'both'];
option_inline_expansion.labels = ['no', 'full-size', 'fit to width',
		'fit to height', 'fit to both'];

function option_high_res() {
}
option_high_res.id = 'nohighres';
option_high_res.label = 'High-res expansions';
option_high_res.type = 'revcheckbox';

$DOC.on('mouseup', 'img, video', function (event) {
	/* Bypass expansion for non-left mouse clicks */
	if (options.get('inlinefit') != 'none' && event.which > 1) {
		var img = $(this);
		img.data('skipExpand', true);
		setTimeout(function () {
			img.removeData('skipExpand');
		}, 100);
	}
});

$DOC.on('click', 'img, video', function (event) {
	if (options.get('inlinefit') != 'none') {
		var $target = $(this);
		if (!$target.data('skipExpand'))
			toggle_expansion($target, event);
	}
});

function toggle_expansion(img, event) {
	var href = img.parent().attr('href');
	if (/^\.\.\/outbound\//.test(href))
		return;
	if (event.metaKey)
		return;
	event.preventDefault();
	var expand = !img.data('thumbSrc');
	if (expand)
		img.closest('figure').addClass('expanded');
	else
		img.closest('figure').removeClass('expanded');
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
	if (fullWidthExpansion)
		contract_full_width(parent_post($img));
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
	var video = /\.webm$/i.test(href);
	var dims = a.siblings('figcaption').text().match(/(\d+)x(\d+)/);
	if (!dims)
		return;
	var tw = $img.width(), th = $img.height();
	var w = parseInt(dims[1], 10), h = parseInt(dims[2], 10);
	// if this is a high-density screen, reduce image size appropriately
	var r = window.devicePixelRatio;
	if (!options.get('nohighres') && !video && r && r > 1) {
		if (w/r > tw && h/r > th) {
			w /= r;
			h /= r;
		}
	}

	$img.remove();
	$img = $(video ? '<video>' : '<img>', {
		src: href,
		width: w, height: h,
		data: {
			thumbWidth: tw, thumbHeight: th,
			thumbSrc: $img.attr('src'),
		},
		prop: video ? {autoplay: true, loop: true} : {},
	}).appendTo(a);

	var fit = options.get('inlinefit');
	if (fit != 'none') {
		var both = fit == 'both';
		fit_to_window($img, w, h, both || fit == 'width',
				both || fit == 'height');
	}
}

function fit_to_window($img, w, h, widthFlag, heightFlag) {
	var $post = parent_post($img);
	var overX = 0, overY = 0;
	if (widthFlag) {
		var innerWidth = $(window).innerWidth();
		var rect = $post.length && $post[0].getBoundingClientRect();
		if ($post.is('article')) {
			if (fullWidthExpansion && w > innerWidth) {
				overX = w - innerWidth;
				expand_full_width($img, $post, rect);
				heightFlag = false;
			}
			else
				overX = rect.right - innerWidth;
		}
		else if ($post.is('section'))
			overX = w - (innerWidth - rect.left*2);
	}
	if (heightFlag) {
		overY = h - ($(window).innerHeight() - 20);
	}

	var aspect = h / w;
	var newW, newH;
	if (overX > 0) {
		newW = w - overX;
		newH = aspect * newW;
	}
	if (overY > 0) {
		// might have to fit to both width and height
		var maybeH = h - overY;
		if (!newH || maybeH < newH) {
			newH = maybeH;
			newW = newH / aspect;
		}
	}

	if (newW > 50 && newH > 50)
		$img.width(newW).height(newH);
}

function expand_full_width($img, $post, rect) {
	var img = $img[0].getBoundingClientRect();
	$img.css('margin-left', -img.left + 'px');
	var over = rect.right - img.right;
	if (over > 0) {
		$post.css({
			'margin-right': -over+'px',
			'padding-right': 0,
			'border-right': 'none',
		});
	}
}

function contract_full_width($post) {
	if ($post.css('margin-right')[0] == '-') {
		$post.css({
			'margin-right': '',
			'padding-right': '',
			'border-right': '',
		});
	}
}

/* SHORTCUT KEYS */

var shortcuts = [
	{label: 'New post', name: 'new', which: 78},
	{label: 'Image spoiler', name: 'togglespoiler', which: 73},
	{label: 'Finish post', name: 'done', which: 83},
];

function toggle_shortcuts(event) {
	event.preventDefault();
	var $shortcuts = $('#shortcuts');
	if ($shortcuts.length)
		return $shortcuts.remove();
	$shortcuts = $('<div/>', {
		id: 'shortcuts',
		click: select_shortcut,
		keyup: change_shortcut,
	});
	shortcuts.forEach(function (s) {
		var value = String.fromCharCode(shortcutKeys[s.name]);
		var $label = $('<label>', {text: s.label});
		$('<input>', {
			id: s.name, maxlength: 1, val: value,
		}).prependTo($label);
		$label.prepend(document.createTextNode('Alt+'));
		$shortcuts.append($label, '<br>');
	});
	$shortcuts.appendTo('#options-panel');
}

function select_shortcut(event) {
	if ($(event.target).is('input'))
		$(event.target).val('');
}

function change_shortcut(event) {
	if (event.which == 13)
		return false;
	var $input = $(event.target);
	var letter = $input.val();
	if (!(/^[a-z]$/i.exec(letter)))
		return;
	var which = letter.toUpperCase().charCodeAt(0);
	var name = $input.attr('id');
	if (!(name in shortcutKeys))
		return;
	shortcutKeys[name] = which;

	var shorts = options.get('shortcuts')
	if (!_.isObject(shorts)) {
		shorts = {};
		shorts[name] = which;
		options.set('shortcuts', shorts);
	}
	else {
		shorts[name] = which;
		options.trigger('change'); // force save
	}

	$input.blur();
}

_.defer(function () {
	load_ident();
	var save = _.debounce(save_ident, 1000);
	function prop() {
		if (postForm)
			postForm.propagate_ident();
		save();
	}
	$name.input(prop);
	$email.input(prop);

	optSpecs.forEach(function (spec) {
		spec.id = spec.id.replace(/\$BOARD/g, BOARD);
	});

	$('<a id="options">Options</a>').click(function () {
		var $opts = $('#options-panel');
		if (!$opts.length)
			$opts = make_options_panel().appendTo('body');
		if ($opts.is(':hidden'))
			oneeSama.trigger('renderOptions', $opts);
		$opts.toggle('fast');
	}).insertAfter('#sync');

	optSpecs.forEach(function (spec) {
		spec(options.get(spec.id));
	});

	var prefs = options.get('shortcuts') || {};
	shortcuts.forEach(function (s) {
		shortcutKeys[s.name] = prefs[s.name] || s.which;
	});
});

function make_options_panel() {
	var $opts = $('<div/>', {"class": 'modal', id: 'options-panel'});
	$opts.change(function (event) {
		var $o = $(event.target), id = $o.attr('id'), val;
		var spec = _.find(optSpecs, function (s) {
			return s.id == id;
		});
		if (!spec)
			return;
		if (spec.type == 'checkbox')
			val = !!$o.prop('checked');
		else if (spec.type == 'revcheckbox')
			val = !$o.prop('checked');
		else if (spec.type == 'positive')
			val = Math.max(parseInt($o.val(), 10), 1);
		else if (spec.type == 'image'){
			var trimmed = $o.val().trim();
			if (/^$|\.(jpe?g|png|gif)$/i.test(trimmed))
				val = trimmed;
		}
		else
			val = $o.val();
		options.set(id, val);
		with_dom(function () {
			spec(val);
		});
	});
	optSpecs.forEach(function (spec) {
		var id = spec.id;
		if (nashi.opts.indexOf(id) >= 0)
			return;
		var val = options.get(id), $input, type = spec.type;
		if (type == 'checkbox' || type == 'revcheckbox') {
			var b = (type == 'revcheckbox') ? !val : val;
			$input = $('<input type="checkbox" />')
				.prop('checked', b ? 'checked' : null);
		}
		else if (type == 'positive') {
			$input = $('<input />', {
				width: '4em',
				maxlength: 4,
				val: val,
			});
		} else if (type == 'image'){
			$input = $('<input />', {
				placeholder: 'Local Image URL',
				val: val
			});
		}
		else if (type instanceof Array) {
			$input = $('<select/>');
			var labels = spec.labels || {};
			type.forEach(function (item, i) {
				var label = labels[i] || item;
				$('<option/>')
					.text(label).val(item)
					.appendTo($input);
			});
			if (type.indexOf(val) >= 0)
				$input.val(val);
		}
		var $label = $('<label/>').attr('for', id).text(spec.label);
		$opts.append($input.attr('id', id), ' ', $label, '<br>');
	});
	if (!nashi.shortcuts) {
		$opts.append($('<a/>', {
			href: '#', text: 'Shortcuts',
			click: toggle_shortcuts,
		}));
	}
	oneeSama.trigger('initOptions', $opts);
	return $opts.hide();
}

})();
