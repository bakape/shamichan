var curPostNum = 0;

var client = new Faye.Client(FAYE_URL, {
	timeout: 60
});

function insert_post(msg) {
	if (msg.num == curPostNum)
		return;
	var post = $(gen_post_html(msg));
	post.insertAfter('ul li[class!=replylink]:last');
}

function insert_reply_box() {
	var link = $('<a>[Reply]</a>')
	link.click(reply_form);
	var box = $('<li class="replylink"/>').append(link);
	$('ul').append(box);
}

function update_post(msg) {
	var body = $('li[name=post' + msg.num + '] blockquote');
	body.append(document.createTextNode(msg.frag));
	body.html(body.html().replace(/\n/g, '<br>'));
}

function finish_post(msg) {
	var post = $('li[name=post' + msg.num + ']');
	post.removeClass('editing');
}

client.subscribe('/thread/new', insert_post);
client.subscribe('/frag', update_post);
client.subscribe('/thread/done', finish_post);

function my_id() {
	/* XXX: temp */
	return Math.floor(Math.random() * 4e15 + 1);
}

function reply_form() {
	var buffer = $('<p/>');
	var meta = $('<span><b/> <code/> <time/></span>');
	var posterName = $('input[name=name]').val().trim();
	var posterEmail = $('input[name=email]').val().trim();
	var input = $('<input name="body" class="trans"/>');
	var blockquote = $('<blockquote/>').append(buffer).append(input);
	var post = $('<li/>').append(meta).append(blockquote);
	var sentAllocRequest = false, allocSubscription = null;
	var myId = my_id();

	var parsed = parse_name(posterName);
	meta.children('b').text(parsed[0]);
	meta.children('code').text(parsed[1] && '!?');
	if (posterEmail) {
		/* TODO: add link */
	}

	function got_allocation(msg) {
		allocSubscription.cancel();
		meta.children('b').text(msg.name);
		meta.children('code').text(msg.trip);
		meta.children('time').text(time_to_str(msg.time));
		curPostNum = msg.num;
		meta.append(' No.' + curPostNum);
		post.addClass('editing');

		var submit = $('<input type="button" value="Done"/>')
		post.append(submit)
		submit.click(function () {
			/* transform into normal post */
			commit(input.val());
			input.remove();
			submit.remove();
			buffer.replaceWith(buffer.contents());
			post.removeClass('editing');

			curPostNum = 0;
			client.publish('/post/done', msg);
			insert_reply_box();
		});
	}
	function commit(text) {
		if (!curPostNum && !sentAllocRequest) {
			client.publish('/post/new', {
				id: myId,
				name: posterName,
				email: posterEmail,
				frag: text
			});
			allocSubscription = client.subscribe('/post/ok/'
					+ myId, got_allocation);
			sentAllocRequest = true;
		}
		else if (curPostNum) {
			client.publish('/post/frag',
				{id: myId, num: curPostNum, frag: text});
		}
		buffer.append(document.createTextNode(text));
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
			commit(input.val() + '\n');
			buffer.append('<br>');
			input.val('');
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
	$(this).parent().replaceWith(post);
	input.focus();
}

$(document).ready(function () {
	insert_reply_box();
});
