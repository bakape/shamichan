var curPostNum = 0;
var activePosts = {};
var liveFeed = true;
var threads = {};
var dispatcher = {};
var THREAD = null;

function send(msg) {
	socket.send(JSON.stringify(msg));
}

function make_reply_box() {
	var box = $('<aside><a>[Reply]</a></aside>');
	box.find('a').click(new_post_form);
	return box;
}

function insert_new_post_boxes() {
	make_reply_box().appendTo('section');
	if (!THREAD) {
		var box = $('<aside><a>[New thread]</a></aside>');
		box.find('a').click(new_post_form);
		$('hr').after(box);
	}
}

function insert_formatted(text, buffer, state) {
	format_fragment(text, state, function (frag) {
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
	});
}

function toggle_live() {
	liveFeed = $(this).attr('checked');
	if (liveFeed)
		$('section').show();
}

dispatcher[INSERT_POST] = function (msg) {
	var post = $(gen_post_html(msg));
	activePosts[msg.num] = post;
	var section = null;
	if (msg.op) {
		section = threads[msg.op];
		section.find('article:last').after(post);
		if (THREAD || !liveFeed)
			return;
		section.detach();
	}
	else {
		var section = $('<section id="thread' + msg.num + '"/>'
				).append(post);
		threads[msg.num] = section;
		if (!curPostNum)
			section.append(make_reply_box());
		if (!liveFeed)
			section.hide();
	}
	/* Insert it at the top; need a more robust way */
	var fencepost = $('body > aside');
	section.insertAfter(fencepost.length ? fencepost : 'hr');
};

dispatcher[UPDATE_POST] = function (msg) {
	var num = msg[0], frag = msg[1], state = [msg[2], msg[3]];
	var post = activePosts[num];
	insert_formatted(frag, post.find('blockquote'), state);
};

dispatcher[FINISH_POST] = function (num) {
	activePosts[num].removeClass('editing');
	delete activePosts[num];
};

function extract_num(q, prefix) {
	return parseInt(q.attr('id').replace(prefix, ''));
}

function new_post_form() {
	var buffer = $('<p/>'), line_buffer = $('<p/>');
	var meta = $('<header><b/> <code/> <time/></header>');
	var nameField = $('input[name=name]');
	var emailField = $('input[name=email]');
	var input = $('<input name="body" class="trans"/>');
	var blockquote = $('<blockquote/>');
	var post = $('<article/>');
	var postOp = null;
	var dummy = $(document.createTextNode(' '));
	var sentAllocRequest = false, unallocatedBuffer = '';
	var thread = $(this).parents('section');
	var state = initial_post_state();
	var INPUT_MIN_SIZE = 2;

	blockquote.append.apply(blockquote, [buffer, line_buffer, input]);
	post.append.apply(post, [meta, blockquote]);

	function propagate_fields() {
		var name = nameField.val().trim();
		var parsed = parse_name(name);
		meta.children('b').text(parsed[0]);
		meta.children('code').text((parsed[1] || parsed[2]) && '!?');
		var email = emailField.val().trim();
		if (email) {
			/* TODO: add link */
		}
	}
	propagate_fields();
	nameField.change(propagate_fields).keypress(propagate_fields);
	emailField.change(propagate_fields).keypress(propagate_fields);

	dispatcher[ALLOCATE_POST] = function (msg) {
		var num = msg.num;
		nameField.unbind();
		emailField.unbind();
		meta.children('b').text(msg.name);
		meta.children('code').text(msg.trip);
		meta.children('time').text(readable_time(msg.time)
				).attr('datetime', datetime(msg.time));
		curPostNum = num;
		meta.append(' <a href="#q' + num + '">No.</a><a href="'
				+ post_url(msg) + '">' + num + '</a>');
		post.attr('id', 'q' + num).addClass('editing');
		if (!postOp) {
			thread.attr('id', 'thread' + num);
			threads[num] = thread;
		}

		var submit = $('<input type="button" value="Done"/>')
		post.append(submit)
		submit.click(function () {
			/* transform into normal post */
			commit(input.val());
			input.remove();
			submit.remove();
			insert_formatted(line_buffer.text(), buffer, state);
			buffer.replaceWith(buffer.contents());
			line_buffer.remove();
			post.removeClass('editing');

			dispatcher[ALLOCATE_POST] = null;
			curPostNum = 0;
			send([FINISH_POST]);
			insert_new_post_boxes();
		});
	};
	function commit(text) {
		if (!text)
			return;
		if (!curPostNum && !sentAllocRequest) {
			var msg = {
				name: nameField.val().trim(),
				email: emailField.val().trim(),
				frag: text
			};
			if (postOp)
				msg.op = postOp;
			send([ALLOCATE_POST, msg]);
			sentAllocRequest = true;
		}
		else if (curPostNum) {
			if (unallocatedBuffer) {
				send(unallocatedBuffer + text);
				unallocatedBuffer = '';
			}
			else
				send(text);
		}
		else
			unallocatedBuffer += text;
		if (text.indexOf('\n') >= 0) {
			var lines = text.split('\n');
			lines[0] = line_buffer.text() + lines[0];
			line_buffer.text(lines.pop());
			for (var i = 0; i < lines.length; i++)
				insert_formatted(lines[i]+'\n', buffer, state);
		}
		else {
			line_buffer.append(document.createTextNode(text));
		}
	}
	function commit_words(text, spaceEntered) {
		var words = text.trim().split(/ +/);
		var endsWithSpace = text.length > 0
				&& text.charAt(text.length-1) == ' ';
		var newWord = endsWithSpace && !spaceEntered;
		if (newWord && words.length > 1) {
			input.val(words.pop() + ' ');
			commit(words.join(' ') + ' ');
		}
		else if (words.length > 2) {
			var last = words.pop();
			input.val(words.pop() + ' ' + last
					+ (endsWithSpace ? ' ' : ''));
			commit(words.join(' ') + ' ');
		}
	}
	input.attr('size', INPUT_MIN_SIZE);
	input.keydown(function (event) {
		var key = event.keyCode;
		if (key == 13) {
			if (sentAllocRequest || input.val().replace(' ', '')) {
				commit(input.val() + '\n');
				input.val('');
			}
			event.preventDefault();
		}
		else {
			commit_words(input.val(), key == 27);
		}
		var cur_size = input.attr('size');
		var right_size = Math.max(Math.round(input.val().length * 1.5),
				INPUT_MIN_SIZE);
		if (cur_size != right_size) {
			input.attr('size', (cur_size + right_size) / 2);
		}
	});
	var parent = $(this).parent()
	if (thread.length) {
		postOp = extract_num(thread, 'thread');
		parent.replaceWith(post);
	}
	else
		thread = $('<section/>').replaceAll(parent).append(post);
	$('aside').remove();
	input.focus();
}

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
	$('#sync').text('Synched.');
}

dispatcher[INVALID] = function (msg) {
	$('#sync').text('Sync error.');
}

$(document).ready(function () {
	$('.editing').each(function(index) {
		var post = $(this);
		activePosts[extract_num(post, 'q')] = post;
	});
	$('section').each(function (index) {
		var section = $(this);
		threads[extract_num(section, 'thread')] = section;
	});
	var m = window.location.pathname.match(/\/(\d+)$/);
	if (m)
		THREAD = parseInt(m[1]);
	m = window.location.hash.match(/^(#q\d+)$/);
	if (m)
		$(m[1]).addClass('highlight');
	insert_new_post_boxes();

	socket.on('connect', on_connect);
	socket.on('disconnect', attempt_reconnect);
	socket.on('message', function (data) {
		msgs = JSON.parse(data);
		for (var i = 0; i < msgs.length; i++) {
			var msg = msgs[i];
			var type = msg.shift();
			dispatcher[type](msg.length == 1 ? msg[0] : msg);
			SYNC++;
		}
	});
	socket.connect();

	$('time').each(function (index) {
		var time = $(this);
		time.text(readable_time(new Date(time.attr('datetime'))));
	});

	if (!THREAD) {
		$('#sync').after($('<span id="live"><label for="live_check">'
			+ 'Real-time bump</label><input type="checkbox" '
			+ 'id="live_check" checked /></span>'));
		$('#live_check').change(toggle_live);
	}
});

var h5s = ['aside', 'article', 'code', 'section', 'time'];
for (var i = 0; i < h5s.length; i++)
	document.createElement(h5s[i]);
