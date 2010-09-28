var curPostNum = 0;
var myPosts = {};
var activePosts = {};

var client = new Faye.Client(FAYE_URL, {
	timeout: 60
});

function make_reply_box() {
	var box = $('<li class="replylink"><a>[Reply]</a></li>');
	box.find('a').click(new_post_form);
	return box;
}

function insert_new_post_boxes() {
	$('ul:not(.newlink)').append(make_reply_box());
	$('hr').after('<ul class="newlink"><li><a>[New thread]</a></li></ul>');
	$('.replylink a, .newlink a').click(new_post_form);
}

function insert_formatted(text, dest, state) {
	for (var i = 0; i < state.spoilers; i++)
		dest = dest.children('del:last');
	var newline = text.indexOf('\n');
	if (newline >= 0) /* max one newline allowed */
		text = text.substr(0, newline);
	var frags = format_line(text, state);
	if (newline >= 0)
		frags.push(safe('<br>'));
	dest.append(flatten(frags).join(''));
}

function is_mine(msg) {
	return (msg.num == curPostNum || msg.num in myPosts);
}

function insert_post(msg) {
	if (is_mine(msg))
		return;
	var state = {spoilers: 0};
	var post = $(gen_post_html(msg, state));
	state.li = post;
	activePosts[msg.num] = state;
	if (msg.op) {
		post.insertAfter('ul[name=thread' + msg.op
				+ '] li:not(.replylink):last');
		return;
	}
	var new_ul = $('<ul name="thread' + msg.num + '" />')
	new_ul.append(post).insertBefore('ul:not(.newlink):first');
	if (!curPostNum)
		new_ul.append(make_reply_box());
}

function update_post(msg) {
	if (is_mine(msg))
		return;
	var post = activePosts[msg.num];
	insert_formatted(msg.frag, post.li.find('blockquote'), post);
}

function finish_post(msg) {
	if (is_mine(msg))
		return;
	activePosts[msg.num].li.removeClass('editing');
	delete activePosts[msg.num];
}

client.subscribe('/thread/new', insert_post);
client.subscribe('/frag', update_post);
client.subscribe('/thread/done', finish_post);

function my_id() {
	/* XXX: temp */
	return Math.floor(Math.random() * 4e15 + 1);
}

function new_post_form() {
	var buffer = $('<p/>'), line_buffer = $('<p/>');
	var meta = $('<span><b/> <code/> <time/></span>');
	var posterName = $('input[name=name]').val().trim();
	var posterEmail = $('input[name=email]').val().trim();
	var input = $('<input name="body" class="trans"/>');
	var blockquote = $('<blockquote/>');
	var post = $('<li/>');
	var postOp = null;
	var dummy = $(document.createTextNode(' '));
	var sentAllocRequest = false, allocSubscription = null;
	var myId = my_id();
	var ul = $(this).parents('ul');
	var state = {spoilers: 0};

	blockquote.append.apply(blockquote, [buffer, line_buffer, input]);
	post.append.apply(post, [meta, blockquote]);

	var parsed = parse_name(posterName);
	meta.children('b').text(parsed[0]);
	meta.children('code').text(parsed[1] && '!?');
	if (posterEmail) {
		/* TODO: add link */
	}

	if (ul.hasClass('newlink'))
		ul.removeClass('newlink');
	else
		postOp = parseInt(ul.attr('name').replace('thread', ''));

	function got_allocation(msg) {
		var num = msg.num;
		allocSubscription.cancel();
		meta.children('b').text(msg.name);
		meta.children('code').text(msg.trip);
		meta.children('time').text(time_to_str(msg.time));
		curPostNum = num;
		myPosts[num] = 1;
		meta.append(' No.' + curPostNum);
		post.addClass('editing');
		post.attr('name', 'q' + num);
		if (!postOp)
			ul.attr('name', 'thread' + num);

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

			curPostNum = 0;
			client.publish('/post/done', {id: myId, num: num});
			insert_new_post_boxes();
		});
	}
	function commit(text) {
		if (!curPostNum && !sentAllocRequest) {
			var msg = {
				id: myId,
				name: posterName,
				email: posterEmail,
				frag: text
			};
			if (postOp) msg.op = postOp;
			client.publish('/post/new', msg);
			allocSubscription = client.subscribe('/post/ok/'
					+ myId, got_allocation);
			sentAllocRequest = true;
		}
		else if (curPostNum) {
			/* TODO: Maybe buffer until allocation okayed? */
			client.publish('/post/frag',
				{id: myId, num: curPostNum, frag: text});
		}
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
	/* do the switch */
	$(this).parent().replaceWith(dummy);
	$('.newlink, .replylink').remove();
	dummy.replaceWith(post);
	input.focus();
}

$(document).ready(function () {
	insert_new_post_boxes();
	$('.editing').each(function(index) {
		var li = $(this);
		var num = parseInt(li.attr('name').replace('q', ''));
		var state = {spoilers: 0, li: li};
		parse_spoilers(li.find('blockquote').html(), state);
		activePosts[num] = state;
	});
});
