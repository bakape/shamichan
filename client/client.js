var BOARD, THREAD, BUMP, PAGE;
var syncs = {}, nonces = {}, ownPosts = {};
var readOnly = ['archive'];
var $name = $('input[name=name]'), $email = $('input[name=email]');
var $ceiling = $('hr:first');
var $sizer = $('<pre></pre>');
var lockedToBottom, lockKeyHeight;
var options, postForm, preview, previewNum;
var inputMinSize = 300, nashi = {opts: []};
var spoilerImages = config.SPOILER_IMAGES;
var spoilerCount = spoilerImages.normal.length + spoilerImages.trans.length;

var connSM = new FSM('load');
var sessionId;

(function () {
	var p = location.pathname;
	BOARD = p.match(/^\/(.+?)\//)[1];
	var t = p.match(/\/(\d+)$/);
	THREAD = t ? parseInt(t[1], 10) : 0;
	BUMP = !!p.match(/\/live$/);
	PAGE = p.match(/\/page(\d+)$/);
	PAGE = PAGE ? parseInt(PAGE[1], 10) : -1;

	nashi.upload = !!$('<input type="file"/>').prop('disabled');
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
})();

function make_reply_box() {
	var box = $('<aside>[<a>Reply</a>]</aside>');
	box.find('a').click(on_make_post);
	return box;
}

function insert_pbs() {
	if (connSM.state == 'out' || postForm || readOnly.indexOf(BOARD) >= 0)
		return;
	if (THREAD ? $('aside').length : $ceiling.next().is('aside'))
		return;
	make_reply_box().appendTo('section');
	if (!nashi.upload && (BUMP || PAGE == 0)) {
		var box = $('<aside>[<a>New thread</a>]</aside>');
		box.find('a').click(on_make_post);
		$ceiling.after(box);
	}
}

var on_make_post = _.wrap(function () {
	var link = $(this);
	postForm = new PostForm(link.parent(), link.parents('section'));
}, with_dom);

function open_post_box(num) {
	var link = $('#' + num);
	if (link[0].tagName.match(/^section$/i))
		link = link.children('aside');
	else
		link = link.siblings('aside');
	on_make_post.call(link.find('a'));
}

var oneeSama = new OneeSama(function (num) {
	if (this.links && num in this.links)
		this.callback(this.post_ref(num, this.links[num]));
	else
		this.callback('>>' + num);
});
oneeSama.full = oneeSama.op = THREAD;

function inject(frag) {
	var dest = this.buffer;
	for (var i = 0; i < this.state[1]; i++)
		dest = dest.children('del:last');
	if (this.state[0] == S_QUOTE)
		dest = dest.children('em:last');
	if (this.strong)
		dest = dest.children('strong:last');
	var out = null;
	if (frag.safe) {
		var m = frag.safe.match(/^<(\w+)>$/);
		if (m)
			out = document.createElement(m[1]);
		else if (frag.safe.match(/^<\/\w+>$/))
			out = '';
	}
	if (out === null)
		out = escape_fragment(frag);
	if (out)
		dest.append(out);
	return out;
}

// TODO: Unify self-updates with OneeSama; this is redundant
function resolve_own_links(links) {
	if (!postForm)
		return;
	postForm.buffer.find('.nope').each(function () {
		var $a = $(this);
		var m = $a.text().match(/^>>(\d+)$/);
		if (!m)
			return;
		var num = m[1], op = links[num];
		if (op) {
			var url = postForm.imouto.post_url(num, op, false);
			$a.attr('href', url).removeAttr('class');
		}
	});
}

function queue_roll(bit) {
	var n = this.allRolls.sent++;
	var info = this.allRolls[n];
	if (!info)
		info = this.allRolls[n] = {};
	info.bit = bit;
	info.$tag = $(this.callback(safe('<strong>')));
	this.strong = true;
	this.callback(info.dice ? readable_dice(bit, info.dice) : bit);
	this.strong = false;
	this.callback(safe('</strong>'));
}

function resolve_rolls(dice) {
	if (!postForm || !postForm.imouto)
		return;
	var rolls = postForm.imouto.allRolls;
	for (var i = 0; i < dice.length; i++) {
		var n = rolls.seen++;
		var info = rolls[n];
		if (!info)
			info = rolls[n] = {};
		info.dice = dice[i];
		if (info.$tag)
			info.$tag.text(readable_dice(info.bit, info.dice));
	}
}

function get_focus() {
	var focus = window.getSelection().focusNode;
	if (focus && focus.tagName && focus.tagName.match(/^blockquote$/i))
		return $(focus).find('textarea');
}

function section_abbrev(section) {
	var stat = section.find('.omit');
	var m = stat.text().match(/(\d+)\D+(\d+)?/);
	if (!m)
		return false;
	return {stat: stat, omit: parseInt(m[1]), img: parseInt(m[2] || 0)};
}

function shift_replies(section) {
	if (THREAD)
		return;
	var shown = section.children('article[id]:not(:has(form))');
	var rem = shown.length;
	if (rem < ABBREVIATED_REPLIES)
		return;
	var $stat, omit = 0, img = 0;
	var info = section_abbrev(section);
	if (info) {
		$stat = info.stat;
		omit = info.omit;
		img = info.img;
	}
	else {
		$stat = $('<span class="omit"></span>');
		section.children('blockquote,form').last().after($stat);
	}
	for (var i = 0; i < shown.length; i++) {
		var cull = $(shown[i]);
		if (rem-- < ABBREVIATED_REPLIES)
			break;
		if (cull.has('figure').length)
			img++;
		omit++;
		cull.remove();
	}
	$stat.text(abbrev_msg(omit, img));
}

function spill_page() {
	if (THREAD)
		return;
	/* Ugh, this could be smarter. */
	var ss = $('body > section[id]:visible');
	for (i = THREADS_PER_PAGE; i < ss.length; i++)
		$(ss[i]).prev('hr').andSelf().hide();

}

var dispatcher = {};
dispatcher[INSERT_POST] = function (msg) {
	var num = msg[0];
	msg = msg[1];
	if (msg.nonce && msg.nonce in nonces) {
		delete nonces[msg.nonce];
		ownPosts[num] = true;
		if (msg.links)
			resolve_own_links(msg.links);
		if (msg.dice)
			resolve_rolls(msg.dice);
		return;
	}
	msg.num = num;
	msg.editing = true;
	var orig_focus = get_focus();
	oneeSama.links = msg.links;
	var section, hr, bump = true;
	if (msg.op) {
		section = $('#' + msg.op);
		if (!section.length)
			return;
		var post = $(oneeSama.mono(msg));
		shift_replies(section);
		section.children('blockquote,.omit,form,article[id]:last'
				).last().after(post);
		if (!BUMP || is_sage(msg.email)) {
			bump = false;
		}
		else {
			hr = section.next();
			section.detach();
			hr.detach();
		}
	}
	else {
		section = $(oneeSama.monomono(msg).join(''));
		hr = $('<hr/>');
		if (!postForm)
			section.append(make_reply_box());
		if (!BUMP) {
			section.hide();
			hr.hide();
		}
	}

	oneeSama.trigger('afterInsert', msg.op ? post : section);
	if (bump) {
		var fencepost = $('body > aside');
		section.insertAfter(fencepost.length ? fencepost : $ceiling
				).after(hr);
		spill_page();
	}
	if (orig_focus)
		orig_focus.focus();
};

dispatcher[MOVE_THREAD] = function (msg) {
	var num = msg[0];
	msg = msg[1];
	msg.num = num;
	var orig_focus = get_focus();
	oneeSama.links = msg.links;

	var section = $(oneeSama.monomono(msg).join(''));
	var hr = $('<hr/>');
	// No make_reply_box since this is archive-only for now
	if (!BUMP) {
		section.hide();
		hr.hide();
	}
	if (msg.replyctr > 0) {
		var omitMsg = abbrev_msg(msg.replyctr, msg.imgctr - 1);
		$('<span class="omit"/>').text(omitMsg).appendTo(section);
	}

	oneeSama.trigger('afterInsert', section);
	var fencepost = $('body > aside');
	section.insertAfter(fencepost.length ? fencepost : $ceiling
			).after(hr);
	spill_page();
	if (orig_focus)
		orig_focus.focus();
};

dispatcher[IMAGE_STATUS] = function (msg) {
	postForm.uploadStatus.text(msg[0]);
};

dispatcher[INSERT_IMAGE] = function (msg) {
	var focus = get_focus();
	var num = msg[0];
	if (num in ownPosts)
		return;
	var hd = $('#' + num + '>header');
	if (hd.length) {
		insert_image(msg[1], hd, false);
		if (focus)
			focus.focus();
	}
};

dispatcher[UPDATE_POST] = function (msg) {
	var num = msg[0], links = msg[4], extra = msg[5];
	var dice = extra ? extra.dice : null;
	if (num in ownPosts) {
		if (links)
			resolve_own_links(links);
		if (dice)
			resolve_rolls(dice);
		return;
	}
	var bq = $('#' + num + '>blockquote');
	if (bq.length) {
		oneeSama.dice = dice;
		oneeSama.links = links || {};
		oneeSama.callback = inject;
		oneeSama.buffer = bq;
		oneeSama.state = [msg[2] || 0, msg[3] || 0];
		oneeSama.fragment(msg[1]);
	}
};

dispatcher[FINISH_POST] = function (msg) {
	var num = msg[0];
	$('#' + num).removeClass('editing');
	delete ownPosts[num];
};

dispatcher[DELETE_POSTS] = function (msg, op) {
	var ownNum = postForm && postForm.num;
	_.each(msg, function (num) {
		if (num === ownNum)
			return postForm.clean_up(true);
		var post = $('#' + num);
		if (post.length)
			post.remove();
		else if (!THREAD) {
			/* post not visible; decrease omit count */
			var info = section_abbrev($('section#' + op));
			if (info && info.omit > 0) {
				/* No way to know if there was an image. Doh */
				var omit = info.omit - 1;
				if (omit > 0)
					info.stat.text(abbrev_msg(omit,
							info.img));
				else
					info.stat.remove();
			}
		}
	});
};

dispatcher[DELETE_THREAD] = function (msg, op) {
	delete syncs[op];
	if (postForm) {
		var num = postForm.num;
		if ((postForm.op || num) == op)
			postForm.clean_up(true);
		if (num == op)
			return;
	}
	$('section#' + op).next('hr').andSelf().remove();
};

dispatcher[EXECUTE_JS] = function (msg) {
	if (THREAD != msg[0])
		return;
	try {
		eval(msg[1]);
	}
	catch (e) {
		/* fgsfds */
	}
};

function extract_num(q) {
	return parseInt(q.attr('id'), 10);
}

function insert_image(info, header, toppu) {
	var fig = $(flatten(oneeSama.gazou(info, toppu)).join(''));
	if (toppu)
		header.before(fig);
	else
		header.after(fig);
}

function PostForm(dest, section) {
	if (section.length) {
		this.thread = section;
		this.op = extract_num(section);
		this.post = $('<article/>');
	}
	else
		this.post = this.thread = $('<section/>');

	this.buffer = $('<p/>');
	this.line_buffer = $('<p/>');
	this.meta = $('<header><a class="emailcancel"><b/></a>' +
			' <time/></header>');
	this.input = $('<textarea name="body" id="trans" rows="1"/>');
	this.submit = $('<input type="button" value="Done"/>');
	this.blockquote = $('<blockquote/>');
	this.pending = '';
	this.line_count = 1;
	this.char_count = 0;
	this.imouto = new OneeSama(function (num) {
		var $s = $('#' + num);
		if (!$s.is('section'))
			$s = $s.parents('section');
		if ($s.is('section'))
			this.callback(this.post_ref(num, extract_num($s)));
		else
			this.callback(safe('<a class="nope">&gt;&gt;' + num
					+ '</a>'));
	});
	this.imouto.callback = inject;
	this.imouto.op = THREAD;
	this.imouto.state = [S_BOL, 0];
	this.imouto.buffer = this.buffer;
	this.imouto.dice = GAME_BOARDS.indexOf(BOARD) >= 0;
	this.imouto.queueRoll = queue_roll;
	this.imouto.allRolls = {sent: 0, seen: 0};

	shift_replies(section);
	var post = this.post;
	this.blockquote.append(this.buffer, this.line_buffer, this.input);
	this.uploadForm = this.make_upload_form();
	this.uploadStatus = this.uploadForm.find('strong');
	post.append(this.meta, this.blockquote, this.uploadForm);

	this.propagate_ident();

	this.input.keydown($.proxy(this, 'on_key_down'));
	var self = this;
	this.input.input(function () {
		self.on_input();
	});

	if (!this.op)
		this.blockquote.hide();
	dest.replaceWith(post);
	if (!this.op)
		post.after('<hr/>');

	$('aside').remove();

	this.resize_input();
	this.input.focus();
}
var PF = PostForm.prototype;

PF.propagate_ident = function () {
	if (this.num)
		return;
	var parsed = parse_name($name.val().trim());
	var meta = this.meta;
	var $b = meta.find('b');
	$b.text(parsed[0] || ANON);
	if (parsed[1] || parsed[2])
		$b.append(' <code>!?</code>');
	var email = $email.val().trim();
	if (is_noko(email))
		email = '';
	var tag = meta.children('a:first');
	if (email)
		tag.attr('href', 'mailto:' + email).attr('class', 'email');
	else
		tag.removeAttr('href').attr('class', 'emailcancel');
}

dispatcher[ALLOCATE_POST] = function (msg) {
	if (postForm)
		postForm.on_allocation(msg[0]);
	else
		send([FINISH_POST]); // Huh. Just tidy up.
};

PF.on_allocation = function (msg) {
	var num = msg.num;
	ownPosts[num] = true;
	this.num = num;
	this.flush_pending();
	var header = $(flatten(oneeSama.atama(msg)).join(''));
	this.meta.replaceWith(header);
	this.meta = header;
	if (this.op)
		this.post.addClass('editing');
	else
		spill_page();
	oneeSama.trigger('afterInsert', this.post);
	this.post.attr('id', num);

	if (msg.image)
		this.insert_uploaded(msg.image);
	else
		this.update_buttons();
	this.submit.click($.proxy(this, 'finish_wrapped'));
	if (this.uploadForm) {
		this.$cancel.remove();
		this.uploadForm.append(this.submit);
	}
	else
		this.blockquote.after(this.submit);
	if (!this.op) {
		this.blockquote.show();
		this.input.focus();
	}
};

PF.on_allocation_wrapped = function (msg) {
	with_dom(_.bind(this.on_allocation, this, msg));
};

function find_time_param(params) {
	if (!params || params.indexOf('t=') < 0)
		return false;
	params = params.split('&');
	for (var i = 0; i < params.length; i++) {
		var pair = '#' + params[i];
		if (pair.match(youtube_time_re))
			return pair;
	}
	return false;
}

PF.on_key_down = function (event) {
	if (lockedToBottom) {
		lockKeyHeight = $DOC.height();
		setTimeout(entryScrollLock, 0);
	}
	switch (event.which) {
	case 83:
		if (event.altKey) {
			this.finish_wrapped();
			event.preventDefault();
		}
		break;
	case 13:
		event.preventDefault();
	case 32:
		var c = event.which == 13 ? '\n' : ' ';
		// predict result
		var input = this.input;
		var val = input.val();
		val = val.slice(0, input[0].selectionStart) + c +
				val.slice(input[0].selectionEnd);
		this.on_input(val);
		break;
	}
};

PF.on_input = function (val) {
	var input = this.input;
	var start = input[0].selectionStart, end = input[0].selectionEnd;
	if (val === undefined)
		val = input.val();

	/* Turn YouTube links into proper refs */
	var changed = false;
	while (true) {
		var m = val.match(youtube_url_re);
		if (!m)
			break;
		/* Substitute */
		var t = m[4] || '';
		t = find_time_param(m[3]) || find_time_param(m[1]) || t;
		var v = '>>>/watch?v=' + m[2] + t;
		var old = m[0].length;
		val = val.substr(0, m.index) + v + val.substr(m.index + old);
		changed = true;
		/* Compensate caret position */
		if (m.index < start) {
			var diff = old - v.length;
			start -= diff;
			end -= diff;
		}
	}
	if (changed)
		input.val(val);

	var nl = val.lastIndexOf('\n');
	if (nl >= 0) {
		var ok = val.substr(0, nl);
		val = val.substr(nl+1);
		input.val(val);
		if (this.sentAllocRequest || ok.match(/[^ ]/))
			this.commit(ok + '\n');
	}
	else {
		var len = val.length;
		var rev = val.split('').reverse().join('');
		var m = rev.match(/^(\s*\S+\s+\S+)\s+(?=\S)/);
		if (m) {
			var lim = len - m[1].length;
			var destiny = val.substr(0, lim);
			this.commit(destiny);
			val = val.substr(lim);
			start -= lim;
			end -= lim;
			input.val(val);
			input[0].setSelectionRange(start, end);
		}
	}

	input.attr('maxlength', MAX_POST_CHARS - this.char_count);
	this.resize_input(val);
};

function entryScrollLock() {
	/* NOPE */
	if (lockedToBottom) {
		/* Special keyup<->down case */
		var height = $DOC.height();
		if (height > lockKeyHeight)
			window.scrollBy(0, height - lockKeyHeight + 1);
	}
}

PF.resize_input = function (val) {
	var input = this.input;
	if (typeof val != 'string')
		val = input.val();

	$sizer.text(val);
	var left = input.offset().left - this.post.offset().left;
	var size = $sizer.width() + INPUT_ROOM;
	size = Math.max(size, inputMinSize - left);
	input.css('width', size + 'px');
};

PF.upload_error = function (msg) {
	/* TODO: Reset allocation if necessary */
	this.$imageInput.attr('disabled', false);
	this.uploadStatus.text(msg);
	this.uploading = false;
	this.update_buttons();
	if (this.uploadForm)
		this.uploadForm.find('input[name=alloc]').remove();
};

PF.upload_complete = function (info) {
	with_dom(_.bind(this.insert_uploaded, this, info));
};

PF.insert_uploaded = function (info) {
	var form = this.uploadForm, op = this.op;
	insert_image(info, form.siblings('header'), !op);
	this.$imageInput.siblings('strong').andSelf().add(this.$cancel
			).remove();
	form.find('#toggle').remove();
	this.flush_pending();
	this.uploading = false;
	this.uploaded = true;
	this.sentAllocRequest = true;
	this.update_buttons();
	/* Stop obnoxious wrap-around-image behaviour */
	this.blockquote.css({
		'margin-left': this.post.find('img').css('margin-right'),
		'padding-left': (info.dims[2] || info.dims[0]) + 'px'
	});

	this.resize_input();
};

function preview_miru(event, num) {
	if (num != previewNum) {
		var post = $('article#' + num);
		if (!post.length)
			return false;
		if (preview)
			preview.remove();
		preview = $('<div class="preview">' + post.html() + '</div>');
	}
	var height = preview.height();
	if (height < 5) {
		preview.hide();
		$(document.body).append(preview);
		height = preview.height();
		preview.detach().show();
	}
	preview.css({left: (event.pageX + 20) + 'px',
		top: (event.pageY - height - 20) + 'px'});
	if (num != previewNum) {
		$(document.body).append(preview);
		previewNum = num;
	}
	return true;
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

var samePage = new RegExp('^(?:' + THREAD + ')?(#\\d+)$');
function click_shita(event) {
	var target = $(event.target);
	var href = target.attr('href');
	if (href && (THREAD || postForm)) {
		var q = href.match(/#q(\d+)/);
		if (q) {
			event.preventDefault();
			with_dom(function () {
				add_ref(parseInt(q[1], 10));
			});
			return;
		}
		if (THREAD) {
			q = href.match(samePage);
			if (q) {
				$('.highlight').removeClass('highlight');
				$(q[1]).addClass('highlight');
				return;
			}
		}
	}
	if (options.inline) {
		var img = target;
		if (img.is('img') && !img.data('skipExpand')) {
			var href = img.parent().attr('href');
			if (href.match(/^\.\.\/outbound\//))
				return;
			var thumb = img.data('thumbSrc');

			with_dom(function () {
			if (thumb) {
				img.replaceWith($('<img>'
					).width(img.data('thumbWidth')
					).height(img.data('thumbHeight')
					).attr('src', thumb));
			}
			else {
				var caption = img.parent().prev().text();
				var dims = caption.match(/(\d+)x(\d+)/);
				var w = parseInt(dims[1],10),
					h = parseInt(dims[2],10),
					r = window.devicePixelRatio;
				if (r && r > 1) {
					w /= r;
					h /= r;
				}
				img.replaceWith($('<img>').data({
					thumbWidth: img.width(),
					thumbHeight: img.height(),
					thumbSrc: img.attr('src')}
					).attr('src',href).width(w).height(h));
			}
			});

			event.preventDefault();
			return;
		}
	}
	if (target.is('cite')) {
		var m = target.text().match(youtube_re);
		var start = 0;
		if (m[2]) {
			var t = m[2].match(youtube_time_re);
			if (t) {
				if (t[1])
					start += parseInt(t[1], 10) * 3600;
				if (t[2])
					start += parseInt(t[2], 10) * 60;
				if (t[3])
					start += parseInt(t[3], 10);
			}
		}
		var $obj = make_video(m[1], null, null, start);
		with_dom(function () {
			target.replaceWith($obj);
		});
		return;
	}
	if (target.is('del')) {
		target.toggleClass('reveal');
		return;
	}
}

function make_video(id, params, dims, start) {
	if (!dims)
		dims = {width: 425, height: 355};
	if (!params)
		params = {allowFullScreen: 'true'};
	params.allowScriptAccess = 'always';
	var query = {version: 3, autohide: 1, showinfo: 0, fs: 1,
		modestbranding: 1};
	if (start)
		query.start = start;
	if (params.autoplay)
		query.autoplay = params.autoplay;
	if (params.loop) {
		query.loop = '1';
		query.playlist = id;
	}

	var bits = [];
	for (var k in query)
		bits.push(encodeURIComponent(k) + '=' +
				encodeURIComponent(query[k]));
	var uri = encodeURI('http://www.youtube.com/v/' + id) + '?' +
			bits.join('&');
	var $obj = $('<object></object>').attr(dims);
	for (var name in params)
		$obj.append($('<param></param>').attr({name: name,
				value: params[name]}));
	$('<embed></embed>').attr(params).attr(dims).attr({src: uri,
		type: 'application/x-shockwave-flash'}).appendTo($obj);
	return $obj;
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

function tsugi() {
	location.href = $('link[rel=next]').prop('href');
}

var $DOC = $(document);
if (window.scrollMaxY !== undefined) {
	function at_bottom() {
		return window.scrollMaxY <= window.scrollY;
	}
}
else {
	function at_bottom() {
		return window.scrollY + window.innerHeight >= $DOC.height();
	}
}

function scroll_shita() {
	var lock = at_bottom();
	if (lock != lockedToBottom)
		set_scroll_locked(lock);
}

function set_scroll_locked(lock) {
	lockedToBottom = lock;
	var ind = $('#lock');
	if (lockedToBottom)
		ind.show();
	else
		ind.hide();
}

function with_dom(func) {
	var lockHeight, locked = lockedToBottom;
	if (locked)
		lockHeight = $DOC.height();
	var ret = func.call(this);
	if (locked) {
		var height = $DOC.height();
		if (height > lockHeight)
			window.scrollBy(0, height - lockHeight + 1);
	}
	return ret;
}

function add_ref(num) {
	/* Make the post form if none exists yet */
	if (!postForm)
		open_post_box(num);
	/* If a >>link exists, put this one on the next line */
	var input = postForm.input;
	var val = input.val();
	if (val.match(/^>>\d+$/)) {
		input.val(val + '\n');
		// XXX: Fix this dumb hack
		postForm.on_input.call(postForm);
		val = input.val();
	}
	input.val(val + '>>' + num);
	input[0].selectionStart = input.val().length;
	postForm.on_input.call(postForm);
	input.focus();
};

PF.make_alloc_request = function (text) {
	var nonce = Math.floor(Math.random() * 1e16) + 1;
	// TODO: Ought to clear out nonces that never arrive eventually
	nonces[nonce] = true;
	var msg = {
		name: $name.val().trim(),
		email: $email.val().trim(),
		nonce: nonce,
	};
	if (text)
		msg.frag = text;
	if (this.op)
		msg.op = this.op;
	return msg;
};

PF.commit = function (text) {
	var lines;
	if (text.indexOf('\n') >= 0) {
		lines = text.split('\n');
		this.line_count += lines.length - 1;
		var breach = this.line_count - MAX_POST_LINES + 1;
		if (breach > 0) {
			for (var i = 0; i < breach; i++)
				lines.pop();
			text = lines.join('\n');
			this.line_count = MAX_POST_LINES;
		}
	}
	var left = MAX_POST_CHARS - this.char_count;
	if (left < text.length)
		text = text.substr(0, left);
	if (!text)
		return;
	this.char_count += text.length;

	/* Either get an allocation or send the committed text */
	if (!this.num && !this.sentAllocRequest) {
		send([ALLOCATE_POST, this.make_alloc_request(text)]);
		this.sentAllocRequest = true;
		this.update_buttons();
	}
	else if (this.num)
		send(text);
	else
		this.pending += text;

	/* Add it to the user's display */
	var line_buffer = this.line_buffer;
	if (lines) {
		lines[0] = line_buffer.text() + lines[0];
		line_buffer.text(lines.pop());
		for (var i = 0; i < lines.length; i++)
			this.imouto.fragment(lines[i] + '\n');
	}
	else {
		line_buffer.append(document.createTextNode(text));
	}
};

PF.flush_pending = function () {
	if (this.pending) {
		send(this.pending);
		this.pending = '';
	}
};

PF.cancel_upload = function () {
	/* XXX: This is a dumb patch-over and it will fail on races */
	if (this.uploading) {
		this.$iframe.remove();
		this.$iframe = $('<iframe src="" name="upload"/></form>');
		this.uploadForm.append(this.$iframe);
		this.upload_error('');
	}
	else
		this.finish_wrapped();
}

PF.finish = function () {
	if (this.num) {
		this.flush_pending();
		this.commit(this.input.val());
		this.input.remove();
		this.submit.remove();
		if (this.uploadForm)
			this.uploadForm.remove();
		this.imouto.fragment(this.line_buffer.text());
		this.buffer.replaceWith(this.buffer.contents());
		this.line_buffer.remove();
		this.blockquote.css({'margin-left': '', 'padding-left': ''});
		send([FINISH_POST]);
		this.clean_up(false);
	}
	else
		this.clean_up(true);
};

PF.finish_wrapped = _.wrap(PF.finish, with_dom);

PF.clean_up = function (remove) {
	if (remove) {
		if (!this.op)
			this.post.next('hr').remove();
		this.post.remove();
	}

	postForm = null;
	insert_pbs();
};

PF.update_buttons = function () {
	var d = this.uploading || (this.sentAllocRequest && !this.num);
	/* Beware of undefined! */
	this.submit.attr('disabled', !!d);
};

PF.prep_upload = function () {
	this.uploadStatus.text('Uploading...');
	this.input.focus();
	this.uploading = true;
	this.update_buttons();
};

PF.make_upload_form = function () {
	var form = $('<form method="post" enctype="multipart/form-data" '
		+ 'action="/img" target="upload">'
		+ '<input type="button" value="Cancel"/>'
		+ '<input type="file" name="image" accept="image/*"/> '
		+ '<a id="toggle">Spoiler</a> <strong/>'
		+ '<input type="hidden" name="spoiler"/>'
		+ '<input type="hidden" name="client_id"/>'
		+ '<iframe src="" name="upload"/></form>');
	form.find('input[name=client_id]').val(sessionId);
	this.$cancel = form.find('input[type=button]').click($.proxy(this,
			'cancel_upload'));
	this.$iframe = form.find('iframe');
	this.$imageInput = form.find('input[name=image]').change(
			on_image_chosen);
	this.$toggle = form.find('#toggle').click($.proxy(this, 'on_toggle'));
	if (nashi.upload) {
		this.$imageInput.hide();
		this.$toggle.hide();
	}
	this.spoiler = 0;
	this.nextSpoiler = Math.floor(Math.random() * spoilerCount);
	return form;
};

PF.on_toggle = function (event) {
	var self = this;
	if (!this.uploading && !this.uploaded) {
		event.preventDefault();
		if (this.spoiler) {
			this.spoiler = 0;
			/* XXX: Removing the style attr is buggy... */
			set_image('pane.png');
			return;
		}
		var imgs = spoilerImages;
		var i = this.nextSpoiler, n = imgs.normal.length;
		this.spoiler = i < n ? imgs.normal[i] : imgs.trans[i - n];
		this.nextSpoiler = (i+1) % spoilerCount;
		set_image('spoil' + this.spoiler + '.png');
	}
	function set_image(path) {
		self.$toggle.css('background-image', 'url("'
				+ config.MEDIA_URL + 'kana/' + path + '")');
	}
};

function on_image_chosen() {
	if (!$(this).val()) {
		postForm.uploadStatus.text('');
		return;
	}
	postForm.prep_upload();
	var form = postForm.uploadForm;
	if (!postForm.num) {
		var alloc = $('<input type="hidden" name="alloc"/>');
		var request = postForm.make_alloc_request(null);
		form.append(alloc.val(JSON.stringify(request)));
	}
	form.find('input[name=spoiler]').val(postForm.spoiler);
	form.submit();
	$(this).attr('disabled', true);
}

function drop_shita(e) {
	e.stopPropagation();
	e.preventDefault();
	var files = e.dataTransfer.files;
	if (files.length != 1) {
		if (files.length > 1)
			alert('Too many files.');
		return;
	}
	if (!postForm) {
		with_dom(function () {
			if (THREAD)
				open_post_box(THREAD);
			else {
				var $s = $(e.target).closest('section');
				if (!$s.length)
					return;
				open_post_box($s.attr('id'));
			}
		});
	}
	else if (postForm.uploading || postForm.uploaded)
		return;

	postForm.prep_upload();
	postForm.$imageInput.attr('disabled', true);

	var fd = new FormData();
	fd.append('image', files[0]);
	fd.append('client_id', sessionId.toFixed());
	if (!postForm.num) {
		var request = postForm.make_alloc_request(null);
		fd.append('alloc', JSON.stringify(request));
	}
	fd.append('spoiler', postForm.spoiler);
	/* Can't seem to jQuery this shit */
	var xhr = new XMLHttpRequest();
	xhr.open('POST', '/img');
	xhr.setRequestHeader('Accept', 'application/json');
	xhr.onreadystatechange = upload_shita;
	xhr.send(fd);
}

function upload_shita() {
	if (this.readyState != 4)
		return;
	if (this.status == 200) {
		var info;
		try {
			info = JSON.parse(this.responseText);
		}
		catch (e) {
			postForm.upload_error("Bad response.");
		}
		postForm[info.func](info.arg);
	}
	else
		postForm.upload_error("Couldn't get response.");
}

function stop_drag(e) {
	e.stopPropagation();
	e.preventDefault();
}

function setup_upload_drop(e) {
	function go(nm, f) { e.addEventListener(nm, f, false); }
	go('dragenter', stop_drag);
	go('dragexit', stop_drag);
	go('dragover', stop_drag);
	go('drop', drop_shita);
}

dispatcher[SYNCHRONIZE] = connSM.feeder('sync');
dispatcher[INVALID] = connSM.feeder('invalid');

connSM.on('synced', function (msg) {
	var dead_threads = msg.length ? msg[0] : []; /* TODO */
	insert_pbs();

	var m = window.location.hash.match(/^#q(\d+)$/);
	if (m) {
		var id = parseInt(m[1], 10);
		if ($('#' + id).hasClass('highlight')) {
			window.location.hash = '#' + id;
			add_ref(id);
		}
	}
});

connSM.on('out', function () {
	if (postForm)
		postForm.finish();
	$('aside').remove();
	$('.editing').removeClass('editing');
});

var toggles = {};
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
}
toggles.preview.label = 'Hover preview';

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

var $opts = $('<div class="modal"/>');
function toggle_opts() {
	$opts.toggle('fast');
}

$(function () {
	$('section').each(function () {
		var s = $(this);
		syncs[s.attr('id')] = parseInt(s.attr('data-sync'));
	});

	try {
		options = JSON.parse(localStorage.options);
	}
	catch (e) { }
	if (!options)
		options = {};

	var m = window.location.hash.match(/^#q?(\d+)$/);
	if (m)
		$('#' + m[1]).addClass('highlight');

	$(document).click(click_shita);
	$('nav input').click(tsugi);
	setup_upload_drop(document.body);

	var ts = $('time'), ti = 0;
	function make_local() {
		if (ti >= ts.length)
			return;
		var t = $(ts[ti++]);
		var d = t.attr('datetime').replace(/-/g, '/'
			).replace('T', ' ').replace('Z', ' GMT');
		t.text(readable_time(new Date(d).getTime()));
		setTimeout(make_local, 0);
	}
	make_local();

	if (!THREAD) {
		/* Insert image omission count (kinda dumb) */
		var ss = $('section'), si = 0;
		function img_omit() {
			if (si >= ss.length)
				return;
			var s = $(ss[si++]);
			var img = parseInt(s.attr('data-imgs')) -
					s.find('img').length;
			if (img > 0) {
				var stat = s.find('.omit');
				var o = stat.text().match(/(\d*)/)[0];
				stat.text(abbrev_msg(parseInt(o), img));
			}
			setTimeout(img_omit, 0);
		}
		img_omit();
	}
	else {
		$('<span id="lock">Locked to bottom</span>'
				).hide().appendTo('body');
		$(document).scroll(scroll_shita);
		scroll_shita();
	}

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
	$(document.body).append($sizer);
	$('<a id="options">Options</a>').click(toggle_opts
			).insertAfter('#sync');
	delete toggles;
});
