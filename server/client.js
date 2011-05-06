var dispatcher = {}, syncs = {};
var THREAD = window.location.pathname.match(/\/(\d+)$/);
THREAD = THREAD ? parseInt(THREAD[1]) : 0;
var nameField = $('input[name=name]'), emailField = $('input[name=email]');
var ceiling = $('hr:first');
var reconnectTimer = null, resetTimer = null, reconnectDelay = 3000;
var options, outOfSync, postForm, preview, previewNum;

var socket = new io.Socket(window.location.domain, {
	port: PORT,
	transports: ['websocket', 'htmlfile', 'xhr-multipart', 'xhr-polling',
		'jsonp-polling']
});

function load_ident() {
	try {
		function load(key, f) {
			if (!f()) {
				var val = localStorage.getItem(key);
				if (val)
					f(val);
			}
		}
		load('name', $.proxy(nameField, 'val'));
		load('email', $.proxy(emailField, 'val'));
	}
	catch (e) {}
}
load_ident();

function save_ident() {
	try {
		function save(key, val) {
			if (typeof val == 'string')
				localStorage.setItem(key, val);
		}
		save('name', nameField.val());
		if (emailField.val() != 'sage')
			save('email', emailField.val());
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
			ceiling.next().is('aside')))
		return;
	make_reply_box().appendTo('section');
	if (!THREAD) {
		var box = $('<aside>[<a>New thread</a>]</aside>');
		box.find('a').click(on_make_post);
		ceiling.after(box);
	}
}

function on_make_post() {
	var link = $(this);
	postForm = new PostForm(link.parent(), link.parents('section'));
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
oneeSama.dirs = DIRS;
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
	if (focus && focus.tagName && focus.tagName.match(/blockquote/i))
		return $(focus).find('textarea');
}

function shift_replies(section) {
	if (THREAD)
		return;
	var shown = section.children('article[id]:not(:has(form))');
	var rem = shown.length;
	if (rem < ABBREV)
		return;
	var stat = section.find('.omit');
	var omit = 0, img = 0;
	if (stat.length) {
		var m = stat.text().match(/(\d+)\D+(\d+)?/);
		omit = parseInt(m[1]);
		img = parseInt(m[2] || 0);
	}
	else {
		stat = $('<span class="omit"></span>');
		section.children('blockquote,form').last().after(stat);
	}
	for (var i = 0; i < shown.length; i++) {
		var cull = $(shown[i]);
		if (rem-- < ABBREV)
			break;
		if (cull.has('figure').length)
			img++;
		omit++;
		cull.remove();
	}
	stat.text(abbrev_msg(omit, img));
}

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
		if (THREAD || !options.live || msg.email == 'sage') {
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
		if (!options.live) {
			section.hide();
			hr.hide();
		}
	}
	if (bump) {
		var fencepost = $('body > aside');
		section.insertAfter(fencepost.length ? fencepost : ceiling
				).after(hr);
	}
	if (orig_focus)
		orig_focus.focus();
};

dispatcher[IMAGE_STATUS] = function (msg) {
	$('input[name=image] + strong').text(msg[0]);
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
	this.meta = $('<header><a class="emailcancel"><b/> <code/></a>' +
			' <time/></header>');
	this.input = $('<textarea name="body" class="trans" rows="1"/>');
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
	post.append(this.meta, this.blockquote, this.uploadForm);

	var prop = $.proxy(this, 'propagate_fields');
	prop();
	nameField.change(prop).keypress(prop);
	emailField.change(prop).keypress(prop);

	this.input.attr('cols', INPUT_MIN_SIZE);
	this.input.attr('maxlength', MAX_POST_CHARS);
	this.input.keydown($.proxy(this, 'on_key'));
	this.input.keyup($.proxy(function (event) {
		if (this.input.val().indexOf('\n') >= 0)
			this.on_key(null);
	}, this));

	if (!this.op)
		this.blockquote.hide();
	dest.replaceWith(post);
	if (!this.op)
		post.after('<hr/>');

	dispatcher[ALLOCATE_POST] = $.proxy(function (msg) {
		this.on_allocation(msg[0]);
	}, this);
	$('aside').remove();
	this.input.focus();
}

PostForm.prototype.propagate_fields = function () {
	var parsed = parse_name(nameField.val().trim());
	var meta = this.meta;
	meta.find('b').text(parsed[0] || ANON);
	meta.find('code').text((parsed[1] || parsed[2]) && '!?');
	var email = emailField.val().trim();
	if (email == 'noko')
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
	nameField.unbind();
	emailField.unbind();
	save_ident();
	var meta = this.meta;
	meta.find('b').text(msg.name || ANON);
	meta.find('code').text(msg.trip);
	var tag = meta.children('a:first');
	if (msg.email)
		tag.attr('href', 'mailto:' + msg.email).attr('class', 'email');
	else
		tag.removeAttr('href').attr('class', 'emailcancel');
	this.post.attr('id', num);
	var head_end = ' ' + num_html(msg);
	if (this.op) {
		this.post.addClass('editing');
		mpmetrics.track('reply', {to: this.op});
	}
	else {
		head_end += expand_html(num);
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

PostForm.prototype.on_key = function (event) {
	var input = this.input;
	if (event && event.which == 13) {
		if (this.sentAllocRequest || input.val().replace(' ', '')) {
			this.commit(input.val() + '\n');
			input.val('');
		}
		if (event.preventDefault)
			event.preventDefault();
	}
	else
		this.commit_words(event && event.which == 27);
	var cur_size = input.attr('cols');
	var right_size = INPUT_MIN_SIZE + Math.round(input.val().length * 2);
	if (cur_size != right_size) {
		input.attr('cols', (cur_size + right_size) / 2);
	}
	input.attr('maxlength', MAX_POST_CHARS - this.char_count);
};

PostForm.prototype.upload_error = function (msg) {
	/* TODO: Reset allocation if necessary */
	$('input[name=image]').attr('disabled', false
			).siblings('strong').text(msg);
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
	this.update_buttons();
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
	if (event.target.tagName.match(/A$/i)) {
		var m = $(event.target).text().match(/>>(\d+)$/);
		if (m && preview_miru(event, parseInt(m[1])))
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
			add_ref(parseInt(q[1]));
			return;
		}
		if (THREAD) {
			q = href.match(samePage);
			if (q) {
				$('.highlight').removeClass('highlight');
				$('article'+q[1]).addClass('highlight');
				return;
			}
		}
	}
	var img = target.filter('img');
	if (img.length && options.inline) {
		var thumb = img.attr('data-thumb-src');
		if (thumb)
			img.attr('src', thumb).removeAttr('data-thumb-src');
		else {
			img.attr('data-thumb-src', img.attr('src'));
			img.attr('src', img.parent().attr('href'));
		}
		img.removeAttr('width').removeAttr('height');
		event.preventDefault();
	}
}

function add_ref(num) {
	mpmetrics.track('add_ref', {num: num});
	/* Make the post form if none exists yet */
	if (!postForm) {
		var link = $('#' + num);
		if (link[0].tagName.match(/section/i))
			link = link.children('aside');
		else
			link = link.siblings('aside');
		on_make_post.call(link.find('a'));
	}
	/* If a >>link exists, put this one on the next line */
	var input = postForm.input;
	if (input.val().match(/^>>\d+$/))
		postForm.on_key.call(postForm, {which: 13});
	input.val(input.val() + '>>' + num);
	input[0].selectionStart = input.val().length;
	postForm.on_key.call(postForm, null);
	input.focus();
};

PostForm.prototype.make_alloc_request = function (text) {
	var msg = {
		name: nameField.val().trim(),
		email: emailField.val().trim(),
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

PostForm.prototype.commit_words = function (spaceEntered) {
	var text = this.input.val();
	var words = text.trim().split(/ +/);
	var endsWithSpace = text.length > 0
			&& text.charAt(text.length-1) == ' ';
	var newWord = endsWithSpace && !spaceEntered;
	if (newWord && words.length > 1) {
		this.input.val(words.pop() + ' ');
		this.commit(words.join(' ') + ' ');
	}
	else if (words.length > 2) {
		var last = words.pop();
		this.input.val(words.pop() + ' ' + last
				+ (endsWithSpace ? ' ' : ''));
		this.commit(words.join(' ') + ' ');
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
		send([FINISH_POST]);
	}
	else {
		if (!this.op)
			this.post.next('hr').remove();
		this.post.remove();
	}

	dispatcher[ALLOCATE_POST] = null;
	postForm = null;
	insert_pbs();
};

PostForm.prototype.make_upload_form = function () {
	var form = $('<form method="post" enctype="multipart/form-data" '
		+ 'action="." target="upload">'
		+ '<input type="button" value="Cancel"/>'
		+ '<input type="file" name="image"/> <strong/>'
		+ '<input type="hidden" name="client_id" value="'
		+ socket.transport.sessionid + '"/> '
		+ '<iframe src="" name="upload"/></form>');
	this.cancel = form.find('input[type=button]').click($.proxy(this,
			'finish'));
	var user_input = this.input;
	var self = this;
	form.find('input[name=image]').change(function () {
		user_input.focus();
		$(this).siblings('strong').text('');
		if (!$(this).val())
			return;
		self.uploading = true;
		self.update_buttons();
		if (!self.num) {
			var alloc = $('<input type="hidden" name="alloc"/>');
			var request = self.make_alloc_request(null);
			form.append(alloc.val(JSON.stringify(request)));
		}
		form.submit();
		$(this).attr('disabled', true);
	});
	return form;
};

PostForm.prototype.update_buttons = function () {
	var d = this.uploading || (this.sentAllocRequest && !this.num);
	/* Beware of undefined! */
	this.submit.add(this.cancel).attr('disabled', !!d);
};

function sync_status(msg, hover) {
	$('#sync').text(msg).attr('class', hover ? 'error' : '');
}

function on_connect() {
	clearTimeout(reconnectTimer);
	if (outOfSync)
		return;
	resetTimer = setTimeout(function (){ reconnectDelay = 3000; }, 9999);
	sync_status('Syncing...', false);
	send([SYNCHRONIZE, syncs, THREAD]);
}

function attempt_reconnect() {
	clearTimeout(resetTimer);
	if (outOfSync)
		return;
	sync_status('Dropped. Reconnecting...', true);
	socket.connect();
	reconnectTimer = setTimeout(attempt_reconnect, reconnectDelay);
	reconnectDelay = Math.min(reconnectDelay * 2, 60000);
}

dispatcher[SYNCHRONIZE] = function (msg) {
	var dead_threads = msg[0]; /* TODO */
	sync_status('Synced.', false);
	insert_pbs();
};

dispatcher[CATCH_UP] = function (msg) {
	syncs[msg[0]] += msg[1];
};

dispatcher[INVALID] = function (msg) {
	msg = msg[0] ? 'Out of sync: ' + msg[0] : 'Out of sync.';
	sync_status(msg, true);
	outOfSync = true;
	socket.disconnect();
	if (postForm)
		postForm.finish();
	$('aside').remove();
	$('.editing').removeClass('editing');
};

$(function () {
	socket.on('connect', on_connect);
	socket.on('disconnect', attempt_reconnect);
	socket.on('message', function (data) {
		msgs = JSON.parse(data);
		for (var i = 0; i < msgs.length; i++) {
			var msg = msgs[i];
			var type = msg.shift();
			/* Pub-sub messages have an extra OP-num entry */
			if (type >= INSERT_POST && type <= INSERT_IMAGE)
				syncs[msg.pop()]++;
			dispatcher[type](msg);
		}
	});
	socket.connect();

	$('section').each(function () {
		var s = $(this);
		syncs[s.attr('id')] = parseInt(s.attr('data-sync'));
	});

	try {
		options = JSON.parse(localStorage.options);
	}
	catch (e) { }
	if (!options)
		options = {live: true};

	var m = window.location.hash.match(/^#q(\d+)$/);
	if (m && $('#' + m[1]).length) {
		var id = parseInt(m[1]);
		window.location.hash = '#' + id;
		$('article#' + id).addClass('highlight');
		setTimeout(function () { add_ref(id); }, 500);
	}
	else {
		m = window.location.hash.match(/^(#\d+)$/);
		if (m)
			$('article' + m[1]).addClass('highlight');
	}

	$(document).click(click_shita);

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

	var opts = $('<div class="modal"/>').hide();
	var bs = {};
	bs.live = function (b) {
		if (b) {
			$('section').show();
			$('hr').show();
		}
	};
	bs.live.label = 'Real-time bump';
	bs.inline = function (b) {};
	bs.inline.label = 'Inline image expansion';
	bs.preview = function (b) {
		if (b)
			$(document).mousemove(hover_shita);
		else
			$(document).unbind('mousemove', hover_shita);
	}
	bs.preview.label = 'Hover preview';

	/* Pre-load options window */
	function opt_change(id, b) {
		return function (event) {
			options[id] = $(this).attr('checked');
			try {
				localStorage.options = JSON.stringify(options);
			}
			catch (e) {}
			b(options[id]);
		};
	}
	for (var id in bs) {
		var b = bs[id];
		$('<input type="checkbox" id="'+id+'" /> <label for="' +
				id + '">' + b.label + '</label><br>'
			).appendTo(opts).change(opt_change(id, b)
			).attr('checked', options[id] ? 'checked' : null);
		b(options[id]);
	}
	$(document.body).append(opts);
	$('<a id="options">Options</a>').click(function () {
		opts.toggle('fast');
	}).insertAfter('#sync');
});
