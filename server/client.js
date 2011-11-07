var THREAD, BUMP, PAGE, syncs = {};
var $name = $('input[name=name]'), $email = $('input[name=email]');
var $ceiling = $('hr:first');
var $sizer = $('<span id="sizer"></span>');
var commit_deferred = false;
var options, outOfSync, postForm, preview, previewNum;

var socket = io.connect('/', {
	port: PORT,
	transports: ['htmlfile', 'xhr-polling', 'jsonp-polling']
});

(function () {
	var p = location.pathname;
	var t = p.match(/\/(\d+)$/);
	THREAD = t ? parseInt(t[1]) : 0;
	BUMP = !!p.match(/\/live$/);
	PAGE = p.match(/\/page(\d+)$/);
	PAGE = PAGE ? parseInt(PAGE[1]) : -1;

	try {
		function load(key, f) {
			if (!f()) {
				var val = localStorage.getItem(key);
				if (val)
					f(val);
			}
		}
		load('name', $.proxy($name, 'val'));
		load('email', $.proxy($email, 'val'));
	}
	catch (e) {}
})();

function save_ident() {
	try {
		function save(key, val) {
			if (typeof val == 'string')
				localStorage.setItem(key, val);
		}
		save('name', $name.val());
		var email = $email.val();
		if (is_noko(email) || !is_sage(email))
			save('email', email);
	}
	catch (e) {}
}

function send(msg) {
	socket.send(JSON.stringify(msg));
}

function make_reply_box() {
	var box = $('<aside>[<a>Reply</a>]</aside>');
	box.find('a').click(on_make_post);
	return box;
}

function insert_pbs() {
	if (outOfSync || postForm || (THREAD ? $('aside').length :
			$ceiling.next().is('aside')))
		return;
	make_reply_box().appendTo('section');
	if (BUMP || PAGE == 0) {
		var box = $('<aside>[<a>New thread</a>]</aside>');
		box.find('a').click(on_make_post);
		$ceiling.after(box);
	}
}

function on_make_post() {
	var link = $(this);
	postForm = new PostForm(link.parent(), link.parents('section'));
}

function open_post_box(num) {
	var link = $('#' + num);
	if (link[0].tagName.match(/^section$/i))
		link = link.children('aside');
	else
		link = link.siblings('aside');
	on_make_post.call(link.find('a'));
}

function make_link(num, op) {
	var p = {num: num, op: op};
	return safe('<a href="' + post_url(p, false) + '">&gt;&gt;'
			+ num + '</a>');
}

var oneeSama = new OneeSama(function (num) {
	if (this.links && num in this.links)
		this.callback(make_link(num, this.links[num]));
	else
		this.callback('>>' + num);
});
oneeSama.media = MEDIA_URL;
oneeSama.full = THREAD;

function inject(frag) {
	var dest = this.buffer;
	for (var i = 0; i < this.state[1]; i++)
		dest = dest.children('del:last');
	if (this.state[0] == 1)
		dest = dest.children('em:last');
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

	if (oneeSama.check)
		oneeSama.check(msg.op ? post : section);
	if (bump) {
		var fencepost = $('body > aside');
		section.insertAfter(fencepost.length ? fencepost : $ceiling
				).after(hr);
		spill_page();
	}
	if (orig_focus)
		orig_focus.focus();
};

dispatcher[IMAGE_STATUS] = function (msg) {
	postForm.uploadStatus.text(msg[0]);
};

dispatcher[INSERT_IMAGE] = function (msg) {
	var focus = get_focus();
	var hd = $('#' + msg[0] + '>header');
	if (hd.length) {
		insert_image(msg[1], hd, false);
		if (focus)
			focus.focus();
	}
};

dispatcher[UPDATE_POST] = function (msg) {
	var bq = $('#' + msg[0] + '>blockquote');
	if (bq.length) {
		oneeSama.links = msg[4] || {};
		oneeSama.callback = inject;
		oneeSama.buffer = bq;
		oneeSama.state = [msg[2] || 0, msg[3] || 0];
		oneeSama.fragment(msg[1]);
	}
};

dispatcher[FINISH_POST] = function (msg) {
	$('#' + msg[0]).removeClass('editing');
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

function extract_num(q) {
	return parseInt(q.attr('id'));
}

function insert_image(info, header, op) {
	header[op?'before':'after']($(flatten(oneeSama.gazou(info)).join('')));
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
		var thread = $('#' + num).parents('*').andSelf().filter(
				'section');
		if (thread.length)
			this.callback(make_link(num, extract_num(thread)));
		else
			this.callback('>>' + num);
	});
	this.imouto.callback = inject;
	this.imouto.state = [0, 0];
	this.imouto.buffer = this.buffer;

	shift_replies(section);
	var post = this.post;
	this.blockquote.append(this.buffer, this.line_buffer, this.input);
	this.uploadForm = this.make_upload_form();
	this.uploadStatus = this.uploadForm.find('strong');
	post.append(this.meta, this.blockquote, this.uploadForm);

	var prop = $.proxy(this, 'propagate_fields');
	prop();
	$name.change(prop).keyup(prop);
	$email.change(prop).keyup(prop);

	this.input.keydown($.proxy(this, 'on_key')
			).bind('paste', $.proxy(this, 'on_paste')
			).keyup($.proxy(this, 'on_key_up'));

	if (!this.op)
		this.blockquote.hide();
	dest.replaceWith(post);
	if (!this.op)
		post.after('<hr/>');

	dispatcher[ALLOCATE_POST] = $.proxy(function (msg) {
		this.on_allocation(msg[0]);
	}, this);
	$('aside').remove();

	this.resize_input();
	this.input.focus();
}

PostForm.prototype.propagate_fields = function () {
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

PostForm.prototype.on_allocation = function (msg) {
	var num = msg.num;
	this.num = num;
	this.flush_pending();
	$name.unbind();
	$email.unbind();
	save_ident();
	var meta = this.meta;
	var $b = meta.find('b');
	$b.text(msg.name || ANON);
	if (msg.trip)
		$b.append(' <code>' + escape_html(msg.trip) + '</code>');
	var tag = meta.children('a:first');
	if (msg.email)
		tag.attr('href', 'mailto:' + msg.email).attr('class', 'email');
	else
		tag.removeAttr('href').attr('class', 'emailcancel');
	if (oneeSama.check)
		oneeSama.check(this.post);
	this.post.attr('id', num);
	var head_end = ' ' + num_html(msg);
	if (this.op) {
		this.post.addClass('editing');
		mpmetrics.track('reply', {to: this.op});
	}
	else {
		head_end += expand_html(num);
		spill_page();
		mpmetrics.track('create', {num: num});
	}
	meta.children('time').text(readable_time(msg.time)
		).attr('datetime', datetime(msg.time)).after(head_end);

	if (msg.image)
		this.upload_complete(msg.image);
	else
		this.update_buttons();
	this.submit.click($.proxy(this, 'finish'));
	if (this.uploadForm) {
		this.cancel.remove();
		this.uploadForm.append(this.submit);
	}
	else
		this.blockquote.after(this.submit);
	if (!this.op) {
		this.blockquote.show();
		this.input.focus();
	}
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

var yasumono = {
	SPACE_KEY: '  ', RETURN_KEY: '\n\n',
	186: ';:', 187: '=+', 188: ',<', 189: '-_', 190: '.>', 191: '/?',
	192: '`~', 219: '[{', 220: '\\|', 221: ']}', 222: '\'"'
};
var ishygddt = ')!@#$%^&*(';
var metamono = [16, 17, 18, 27, 91];

function predict_result(event, start, end, val) {
	if (!event || typeof start != 'number' || typeof end != 'number')
		return val;
	var which = event.which;
	if (!which || (which>32 && which<41) || metamono.indexOf(which) >= 0)
		return val;
	if (start == end) {
		if (which == BKSP_KEY)
			start--;
		else if (which == DEL_KEY)
			end++;
	}
	var before = val.substr(0, start), after = val.substr(end), mid = '';
	if (which >= 48 && which < 90) {
		mid = String.fromCharCode(which);
		if (!event.shiftKey)
			mid = mid.toLowerCase();
		else if (which < 58)
			mid = ishygddt[which - 48];
	}
	else if (which in yasumono)
		mid = yasumono[which][event.shiftKey ? 1 : 0];
	else if (which != BKSP_KEY && which != DEL_KEY)
		mid = 'W'; /* guess */
	return before + mid + after;
}

PostForm.prototype.on_shortcut = function (event) {
	if (event.which == 83 && event.altKey)
		this.finish();
	else
		return false;
	return true;
};

PostForm.prototype.on_key = function (event) {
	if (event && event.preventDefault && this.on_shortcut(event)) {
		event.preventDefault();
		return;
	}
	var input = this.input;
	var val = input.val();
	var start = input[0].selectionStart, end = input[0].selectionEnd;

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

	var prediction = predict_result(event, start, end, val);

	var nl = prediction.lastIndexOf('\n');
	if (event && nl >= 0) {
		var ok = val.substr(0, nl);
		val = val.substr(nl);
		prediction = prediction.substr(nl);
		input.val(val);
		if (this.sentAllocRequest || ok.match(/[^ ]/))
			this.commit(ok + '\n');
		if (event.preventDefault)
			event.preventDefault();
	}
	else {
		var len = prediction.length;
		var revPred = prediction.split('').reverse().join('');
		var m = revPred.match(/^(\s*\S+\s+\S+)\s+(?=\S)/);
		if (m) {
			var lim = len - m[1].length, keep = len - m[1].length;
			var destiny = val.substr(0, lim);
			if (destiny == prediction.substr(0, lim)) {
				/* Prefix of existing and prediction match */
				this.commit(destiny + ' ');
				val = val.substr(keep);
				prediction = prediction.substr(keep);
				this.input.val(val);
				this.input[0].selectionStart = start - lim;
				this.input[0].selectionEnd = end - lim;
			}
			else {
				/* Ambiguous case. Wait until key-up */
				commit_deferred = true;
			}
		}
	}

	input.attr('maxlength', MAX_POST_CHARS - this.char_count);
	this.resize_input(prediction);
};

PostForm.prototype.on_key_up = function (event) {
	if (commit_deferred)
		this.on_key(null);
	commit_deferred = false;
};

PostForm.prototype.resize_input = function (val) {
	var input = this.input;
	if (typeof val != 'string')
		val = input.val();

	$sizer.text(val);
	var left = input.offset().left - this.post.offset().left;
	var size = $sizer.width() + INPUT_ROOM;
	size = Math.max(size, INPUT_MIN_SIZE - left);
	input.css('width', size + 'px');
};

PostForm.prototype.on_paste = function (event) {
	setTimeout($.proxy(this, 'on_key'), 0);
};

PostForm.prototype.upload_error = function (msg) {
	/* TODO: Reset allocation if necessary */
	$('input[name=image]').attr('disabled', false);
	this.uploadStatus.text(msg);
	this.uploading = false;
	this.update_buttons();
};

PostForm.prototype.upload_complete = function (info) {
	var form = this.uploadForm, op = this.op;
	insert_image(info, form.siblings('header'), !op);
	form.find('input[name=image]').siblings('strong').andSelf().add(
			this.cancel).remove();
	mpmetrics.track('image', op ? {op: op} : {});
	this.flush_pending();
	this.uploading = false;
	this.uploaded = true;
	this.update_buttons();
	/* Stop obnoxious wrap-around-image behaviour */
	this.blockquote.css({
		'margin-left': this.post.find('img').css('margin-right'),
		'padding-left': info.dims[2] + 'px'
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
	preview.css({left: event.pageX + 'px', top: (event.pageY+30) + 'px'});
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

var samePage = new RegExp('^' + THREAD + '(#\\d+)$');
function click_shita(event) {
	var target = $(event.target);
	var href = target.attr('href');
	if (href && (THREAD || postForm)) {
		var q = href.match(/#q(\d+)/);
		if (q) {
			event.preventDefault();
			add_ref(parseInt(q[1], 10));
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
	var img = target.filter('img');
	if (img.length && options.inline && !img.data('skipExpand')) {
		var thumb = img.data('thumbSrc');
		if (thumb) {
			img.width(img.data('thumbWidth')
				).height(img.data('thumbHeight')
				).attr('src', thumb
				).removeData('thumbSrc');
		}
		else {
			var caption = img.parent().prev().text();
			var dims = caption.match(/(\d+)x(\d+)/);
			img.data('thumbWidth', img.width()
				).data('thumbHeight', img.height()
				).data('thumbSrc', img.attr('src')
				).attr('src', img.parent().attr('href')
				).width(dims[1]).height(dims[2]);
		}
		event.preventDefault();
		return;
	}
	if (target.filter('cite').length) {
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
		var url = encodeURI('http://www.youtube.com/v/' + m[1] +
			'?version=3&autohide=1&showinfo=0&fs=1&' +
			'modestbranding=1' + (start ? '&start=' + start : ''));
		var dims = {width: 425, height: 355};
		var params = {allowScriptAccess: 'always',
				allowFullScreen: 'true'};
		var $obj = $('<object></object>').attr(dims);
		for (var name in params)
			$obj.append($('<param></param>').attr({name: name,
					value: params[name]}));
		var $embed = $('<embed></embed>').attr(params).attr({src: url,
			type: 'application/x-shockwave-flash'}).attr(dims);
		$obj.append($embed);
		target.replaceWith($obj);
	}
}

function mouseup_shita(event) {
	/* Bypass expansion for non-left mouse clicks */
	if (options.inline && event.which > 1) {
		var img = $(event.target).filter('img');
		if (img.length) {
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

function add_ref(num) {
	mpmetrics.track('add_ref', {num: num});
	/* Make the post form if none exists yet */
	if (!postForm)
		open_post_box(num);
	/* If a >>link exists, put this one on the next line */
	var input = postForm.input;
	if (input.val().match(/^>>\d+$/))
		postForm.on_key.call(postForm, {which: RETURN_KEY});
	input.val(input.val() + '>>' + num);
	input[0].selectionStart = input.val().length;
	postForm.on_key.call(postForm, null);
	input.focus();
};

PostForm.prototype.make_alloc_request = function (text) {
	var msg = {
		name: $name.val().trim(),
		email: $email.val().trim(),
	};
	if (text)
		msg.frag = text;
	if (this.op)
		msg.op = this.op;
	return msg;
};

PostForm.prototype.commit = function (text) {
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

PostForm.prototype.flush_pending = function () {
	if (this.pending) {
		send(this.pending);
		this.pending = '';
	}
};

PostForm.prototype.finish = function () {
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

PostForm.prototype.clean_up = function (remove) {
	if (remove) {
		if (!this.op)
			this.post.next('hr').remove();
		this.post.remove();
	}

	dispatcher[ALLOCATE_POST] = null;
	postForm = null;
	insert_pbs();
};

PostForm.prototype.update_buttons = function () {
	var d = this.uploading || (this.sentAllocRequest && !this.num);
	/* Beware of undefined! */
	this.submit.add(this.cancel).attr('disabled', !!d);
};

PostForm.prototype.prep_upload = function () {
	this.uploadStatus.text('Sending...');
	this.input.focus();
	this.uploading = true;
	this.update_buttons();
};

PostForm.prototype.make_upload_form = function () {
	var form = $('<form method="post" enctype="multipart/form-data" '
		+ 'action="/img" target="upload">'
		+ '<input type="button" value="Cancel"/>'
		+ '<input type="file" name="image"/> <strong/>'
		+ '<input type="hidden" name="client_id" value="'
		+ socket.socket.sessionid + '"/> '
		+ '<iframe src="" name="upload"/></form>');
	this.cancel = form.find('input[type=button]').click($.proxy(this,
			'finish'));
	form.find('input[name=image]').change(on_image_chosen);
	return form;
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
		if (THREAD)
			open_post_box(THREAD);
		else {
			var $s = $(e.target).closest('section');
			if (!$s.length)
				return;
			open_post_box($s.attr('id'));
		}
	}
	else if (postForm.uploading || postForm.uploaded)
		return;

	postForm.prep_upload();
	postForm.uploadForm.find('input[name=image]').attr('disabled', true);

	var fd = new FormData();
	fd.append('image', files[0]);
	fd.append('client_id', socket.socket.sessionid);
	if (!postForm.num) {
		var request = postForm.make_alloc_request(null);
		fd.append('alloc', JSON.stringify(request));
	}
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
		try {
			var info = JSON.parse(this.responseText);
			postForm[info.func](info.arg);
		}
		catch (e) {
			postForm.upload_error("Bad response.");
		}
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

function sync_status(msg, hover) {
	$('#sync').text(msg).attr('class', hover ? 'error' : '');
}

var MIRU = {};
MIRU.connecting = function () {
	sync_status('Connecting...', false);
};

MIRU.connect = function () {
	sync_status('Syncing...', false);
	send([SYNCHRONIZE, syncs, BUMP]);
};

MIRU.disconnect = function () {
	sync_status('Dropped. Reconnecting...', true);
};

MIRU.reconnect_failed = function () {
	sync_status('Dropped.', true);
};

MIRU.message = function (data) {
	msgs = JSON.parse(data);
	for (var i = 0; i < msgs.length; i++) {
		var msg = msgs[i];
		var type = msg.shift();
		/* Pub-sub messages have an extra OP-num entry */
		var op;
		if (is_pubsub(type)) {
			op = msg.pop();
			syncs[op]++;
		}
		dispatcher[type](msg, op);
	}
};

function socket_miru(setup) {
	var f = $.proxy(socket, setup ? 'on' : 'removeAllListeners');
	for (var ku in MIRU)
		f(ku, MIRU[ku]);
}

dispatcher[SYNCHRONIZE] = function (msg) {
	var dead_threads = msg.length ? msg[0] : []; /* TODO */
	sync_status('Synced.', false);
	insert_pbs();

	var m = window.location.hash.match(/^#q(\d+)$/);
	if (m) {
		var id = parseInt(m[1], 10);
		if ($('#' + id).hasClass('highlight')) {
			window.location.hash = '#' + id;
			add_ref(id);
		}
	}
};

dispatcher[CATCH_UP] = function (msg) {
	syncs[msg[0]] += msg[1];
};

dispatcher[INVALID] = function (msg) {
	msg = msg[0] ? 'Out of sync: ' + msg[0] : 'Out of sync.';
	sync_status(msg, true);
	outOfSync = true;
	socket_miru(false);
	socket.disconnect();
	if (postForm)
		postForm.finish();
	$('aside').remove();
	$('.editing').removeClass('editing');
};

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
	socket_miru(true);

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

	for (var id in toggles) {
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
