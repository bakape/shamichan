var postForm = null;
var activePosts = {};
var liveFeed = true;
var threads = {};
var dispatcher = {};
var THREAD = 0;
var nameField, emailField;
var INPUT_MIN_SIZE = 10;
var ceiling;

function send(msg) {
	socket.send(JSON.stringify(msg));
}

function make_reply_box() {
	var box = $('<aside><a>[Reply]</a></aside>');
	box.find('a').click(PostForm);
	return box;
}

function insert_new_post_boxes() {
	make_reply_box().appendTo('section');
	if (!THREAD) {
		var box = $('<aside><a>[New thread]</a></aside>');
		box.find('a').click(PostForm);
		ceiling.after(box);
	}
}

function make_link(num, op) {
	var p = {num: num, op: op};
	return safe('<a href="' + post_url(p, false) + '">&gt;&gt;'
			+ num + '</a>');
}

function format_link(num, env) {
	if (env.links && num in env.links)
		env.callback(make_link(num, env.links[num]));
	else
		env.callback('>>' + num);
}

function insert_formatted(text, buffer, state, env) {
	if (!env.format_link)
		env.format_link = format_link;
	env.callback = function (frag) {
		var dest = buffer;
		for (var i = 0; i < state[1]; i++)
			dest = dest.children('del:last');
		if (state[0] == 1)
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
	};
	format_fragment(text, state, env);
}

function toggle_live() {
	liveFeed = $(this).attr('checked');
	if (liveFeed)
		$('section').show();
}

dispatcher[INSERT_POST] = function (msg) {
	msg = msg[0];
	if (postForm && msg.num == postForm.num)
		return true;
	msg.format_link = format_link;
	var post = $(gen_post_html(msg, msg));
	post.find('header > a:last').click(add_ref);
	activePosts[msg.num] = post;
	var section = null;
	if (msg.op) {
		section = threads[msg.op];
		section.find('article[id]:last').after(post);
		if (THREAD || !liveFeed)
			return true;
		section.detach();
	}
	else {
		var section = $('<section id="thread' + msg.num + '"/>'
				).append(post).after('<hr/>');
		threads[msg.num] = section;
		if (!postForm)
			section.append(make_reply_box());
		if (!liveFeed)
			section.hide();
	}
	/* Insert it at the top; need a more robust way */
	var fencepost = $('body > aside');
	section.insertAfter(fencepost.length ? fencepost : ceiling);
	return true;
};

dispatcher[UPDATE_POST] = function (msg) {
	var num = msg[0], frag = msg[1], state = [msg[2], msg[3]];
	var env = msg[4] || {};
	var post = activePosts[num];
	if (post)
		insert_formatted(frag, post.find('blockquote'), state, env);
	else
		console.log("Tried to update inactive post #" + num
				+ " with " + JSON.stringify(msg));
	return true;
};

dispatcher[FINISH_POST] = function (msg) {
	num = msg[0];
	activePosts[num].removeClass('editing');
	delete activePosts[num];
	return true;
};

function extract_num(q, prefix) {
	var a = q.attr('id');
	return parseInt(prefix ? a.replace(prefix, '') : a);
}

function upload_error(msg) {
	var msg = $('<strong/>').text(msg);
	$('input[name=image]').attr('disabled', false).after(msg);
}

function upload_complete(info) {
	var form = postForm.uploadForm;
	var metadata = $(flatten(image_metadata(info)).join(''));
	form.siblings('header').append(metadata).after(thumbnail_html(info));
	form.find('input[name=image]').remove();
}

function propagate_fields() {
	var parsed = parse_name(nameField.val().trim());
	postForm.meta.children('b').text(parsed[0] || ANON);
	postForm.meta.children('code').text((parsed[1] || parsed[2]) && '!?');
	var email = emailField.val().trim();
	if (email) {
		/* TODO: add link */
	}
}

var format_env = {format_link: function (num, env) {
	var post = $('#' + num);
	if (post.length) {
		var thread = extract_num(post.parent(), 'thread');
		env.callback(make_link(num, thread));
	}
	else
		env.callback('>>' + num);
}};

function PostForm(link_clicked) {
	if (!(this instanceof PostForm)) {
		return new PostForm(this);
	}
	postForm = this;
	this.buffer = $('<p/>');
	this.line_buffer = $('<p/>');
	this.meta = $('<header><b/> <code/> <time/></header>');
	this.input = $('<textarea name="body" class="trans" rows="1"/>');
	this.uploadForm = null;
	this.submit = $('<input type="button" value="Done"/>');
	this.blockquote = $('<blockquote/>');
	var post = $('<article/>');
	this.post = post;
	this.sentAllocRequest = false;
	this.unallocatedBuffer = '';
	this.state = initial_post_state();

	var input_field = [this.buffer, this.line_buffer, this.input];
	this.blockquote.append.apply(this.blockquote, input_field);
	var post_parts = [this.meta, this.blockquote];
	if (IMAGE_UPLOAD) {
		this.uploadForm = this.make_upload_form();
		post_parts.push(this.uploadForm);
	}
	post.append.apply(post, post_parts);

	propagate_fields();
	nameField.change(propagate_fields).keypress(propagate_fields);
	emailField.change(propagate_fields).keypress(propagate_fields);

	this.input.attr('cols', INPUT_MIN_SIZE);
	this.input.keydown($.proxy(this.on_key, this));
	this.input.keyup($.proxy(function (event) {
		if (this.input.val().indexOf('\n') >= 0)
			this.on_key(null);
	}, this));
	var link = $(link_clicked);
	var parent = link.parent(), section = link.parents('section');
	if (section.length) {
		this.thread = section;
		this.op = extract_num(section, 'thread');
		parent.replaceWith(post);
	}
	else {
		this.thread = $('<section/>').replaceAll(parent).append(post
				).after('<hr/>');
		post.addClass('op');
	}
	dispatcher[ALLOCATE_POST] = $.proxy(function (msg) {
		this.on_allocation(msg[0]);
		/* We've already received a SYNC for this insert */
		return false;
	}, this);
	$('aside').remove();
	this.input.focus();
}

PostForm.prototype.on_allocation = function (msg) {
	var num = msg.num;
	this.num = num;
	nameField.unbind();
	emailField.unbind();
	var meta = this.meta;
	meta.children('b').text(msg.name || ANON);
	meta.children('code').text(msg.trip);
	meta.children('time').text(readable_time(msg.time)
		).attr('datetime', datetime(msg.time)
		).after(' ' + num_html(msg));
	meta.find('a:last').click(add_ref);
	this.post.attr('id', '' + num).addClass('editing');
	if (!this.op) {
		this.thread.attr('id', 'thread' + num);
		threads[num] = this.thread;
	}

	this.submit.attr('disabled', false);
	if (this.uploadForm)
		this.uploadForm.append(this.submit);
	else
		this.blockquote.after(this.submit);
	this.submit.click($.proxy(this.finish, this));
	if (msg.image)
		upload_complete(msg.image);
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
};

function add_ref(num) {
	if (!parseInt(num)) {
		if (!THREAD && !postForm)
			return;
		var post = $(this).parents('article');
		if (!post.length)
			return;
		if (num.preventDefault)
			num.preventDefault();
		num = extract_num(post, false);
	}
	if (!postForm) {
		var link = $('#' + num).siblings('aside').find('a');
		new PostForm(link);
	}
	if (postForm.input.val().match(/^>>\d+$/))
		postForm.on_key.call(postForm, {which: 13});
	postForm.input.val(postForm.input.val() + '>>' + num);
	postForm.on_key.call(postForm, null);
	postForm.input.focus();
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
	if (!text)
		return;
	if (!this.num && !this.sentAllocRequest) {
		send([ALLOCATE_POST, this.make_alloc_request(text)]);
		this.sentAllocRequest = true;
	}
	else if (this.num) {
		if (this.unallocatedBuffer) {
			send(this.unallocatedBuffer + text);
			this.unallocatedBuffer = '';
		}
		else
			send(text);
	}
	else
		this.unallocatedBuffer += text;

	var line_buffer = this.line_buffer;
	if (text.indexOf('\n') >= 0) {
		var lines = text.split('\n');
		lines[0] = line_buffer.text() + lines[0];
		line_buffer.text(lines.pop());
		for (var i = 0; i < lines.length; i++)
			insert_formatted(lines[i] + '\n', this.buffer,
					this.state, format_env);
	}
	else {
		line_buffer.append(document.createTextNode(text));
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
	this.commit(this.input.val());
	this.input.remove();
	this.submit.remove();
	if (this.uploadForm)
		this.uploadForm.remove();
	var buffer = this.buffer, line_buffer = this.line_buffer;
	insert_formatted(line_buffer.text(), buffer, this.state, format_env);
	buffer.replaceWith(buffer.contents());
	line_buffer.remove();
	this.post.removeClass('editing');

	dispatcher[ALLOCATE_POST] = null;
	postForm = null;
	send([FINISH_POST]);
	insert_new_post_boxes();
};

PostForm.prototype.make_upload_form = function () {
	var form = $('<form method="post" enctype="multipart/form-data" '
		+ 'action="." target="upload">'
		+ '<input type="file" name="image"/>'
		+ '<input type="hidden" name="client_id" value="'
		+ socket.transport.sessionid + '"/>'
		+ '<iframe src="" name="upload"/></form>');
	var user_input = this.input;
	form.find('input[name=image]').change(function () {
		user_input.focus();
		$(this).siblings('strong').remove();
		if (!$(this).val())
			return;
		if (!postForm.sentAllocRequest) {
			postForm.submit.attr('disabled', true);
			var alloc = $('<input type="hidden" name="alloc"/>');
			var request = postForm.make_alloc_request(null);
			form.append(alloc.val(JSON.stringify(request)));
		}
		form.submit();
		$(this).attr('disabled', true);
	});
	return form;
};

var socket = new io.Socket(HOST, {
	port: PORT,
	transports: ['websocket', 'htmlfile', 'xhr-multipart', 'xhr-polling',
		'jsonp-polling']
});

var reconnect_timer = null, reset_timer = null, reconnect_delay = 3000;
function on_connect() {
	clearTimeout(reconnect_timer);
	reset_timer = setTimeout(function (){ reconnect_delay = 3000; }, 9999);
	$('#sync').text('Synching...');
	send([SYNCHRONIZE, SYNC, THREAD]);
}

function attempt_reconnect() {
	clearTimeout(reset_timer);
	$('#sync').text('Not synched.');
	socket.connect();
	reconnect_timer = setTimeout(attempt_reconnect, reconnect_delay);
	reconnect_delay = Math.min(reconnect_delay * 2, 60000);
}

dispatcher[SYNCHRONIZE] = function (msg) {
	SYNC = msg[0];
	$('#sync').text('Synched.');
	return false;
}

dispatcher[INVALID] = function (msg) {
	$('#sync').text('Sync error.');
	return false;
}

$(document).ready(function () {
	nameField = $('input[name=name]');
	emailField = $('input[name=email]');
	ceiling = $('hr:first');

	$('.editing').each(function(index) {
		var post = $(this);
		activePosts[extract_num(post, false)] = post;
	});
	$('section').each(function (index) {
		var section = $(this);
		threads[extract_num(section, 'thread')] = section;
	});
	var m = window.location.pathname.match(/\/(\d+)$/);
	if (m)
		THREAD = parseInt(m[1]);
	insert_new_post_boxes();
	m = window.location.hash.match(/^#q(\d+)$/);
	if (m && $('#' + m[1]).length) {
		var id = m[1];
		window.location.hash = '#' + id;
		$('#' + id).addClass('highlight');
		setTimeout(function () { add_ref(id); }, 500);
	}
	else {
		m = window.location.hash.match(/^(#\d+)$/);
		if (m)
			$(m[1]).addClass('highlight');
	}

	socket.on('connect', on_connect);
	socket.on('disconnect', attempt_reconnect);
	socket.on('message', function (data) {
		msgs = JSON.parse(data);
		for (var i = 0; i < msgs.length; i++) {
			var msg = msgs[i];
			var type = msg.shift();
			if (dispatcher[type](msg))
				SYNC++;
		}
	});
	socket.connect();

	$('time').each(function (index) {
		var time = $(this);
		time.text(readable_time(new Date(time.attr('datetime'))));
	});
	$('article header').each(function (index) {
	       $(this).children('a').click(add_ref);
	});

	if (!THREAD) {
		$('#sync').after($('<span id="live"><label for="live_check">'
			+ 'Real-time bump</label><input type="checkbox" '
			+ 'id="live_check" checked /></span>'));
		$('#live_check').change(toggle_live);
	}
});

var h5s = ['abbr', 'aside', 'article', 'code', 'section', 'time'];
for (var i = 0; i < h5s.length; i++)
	document.createElement(h5s[i]);
