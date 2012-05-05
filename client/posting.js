var postForm;
var spoilerImages = config.SPOILER_IMAGES;
var spoilerCount = spoilerImages.normal.length + spoilerImages.trans.length;

connSM.on('synced', postSM.feeder('sync'));
connSM.on('dropped', postSM.feeder('desync'));
connSM.on('desynced', postSM.feeder('desync'));

postSM.act('* + desync -> none', function () {
	if (postForm) {
		postForm.post.removeClass('editing');
		postForm.input.val('');
		postForm.finish();
		postForm = null;
	}
	$('aside').remove();
});

postSM.act('none + sync, draft, alloc + done -> ready', function () {
	if (postForm) {
		postForm.remove();
		postForm = null;
	}
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

postSM.act('ready + new -> draft', function (aside) {
	postForm = new PostForm(aside, aside.parents('section'));
});

postSM.act('draft + alloc -> alloc', function (msg) {
	postForm.on_allocation(msg);
});

$DOC.on('click', 'aside a', _.wrap(function () {
	postSM.feed('new', $(this).parent());
}, with_dom));

function open_post_box(num) {
	var a = $('#' + num);
	postSM.feed('new', a.is('section')
			? a.children('aside') : a.siblings('aside'));
}

function make_reply_box() {
	return $('<aside class="act"><a>Reply</a></aside>');
}

function insert_pbs() {
	if (readOnly.indexOf(BOARD) >= 0)
		return;
	if (THREAD ? $('aside').length : $ceiling.next().is('aside'))
		return;
	make_reply_box().appendTo('section');
	if (!nashi.upload && (BUMP || PAGE == 0))
		$ceiling.after('<aside class="act"><a>New thread</a></aside>');
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
	this.meta = $('<header><a class="nope"><b/></a>' +
			' <time/></header>');
	this.input = $('<textarea name="body" id="trans" rows="1"/>');
	this.submit = $('<input type="button" value="Done"/>');
	this.blockquote = $('<blockquote/>');
	this.$sizer = $('<pre/>').appendTo('body');
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
	oneeSama.trigger('imouto', this.imouto);

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
		tag.removeAttr('href').attr('class', 'nope');
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

PF.on_image_alloc = function (msg) {
	with_dom(function () {
		postSM.feed('alloc', msg);
	});
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

PF.resize_input = function (val) {
	var input = this.input;
	if (typeof val != 'string')
		val = input.val();

	this.$sizer.text(val);
	var left = input.offset().left - this.post.offset().left;
	var size = this.$sizer.width() + INPUT_ROOM;
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
};

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
		this.preserve = true;
	}
	postSM.feed('done');
};

PF.finish_wrapped = _.wrap(PF.finish, with_dom);

PF.remove = function () {
	if (!this.preserve) {
		if (!this.op)
			this.post.next('hr').remove();
		this.post.remove();
	}
	this.$sizer.remove();
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
		+ '<input type="button" id="toggle"> <strong/>'
		+ '<input type="hidden" name="spoiler"/>'
		+ '<input type="hidden" name="client_id"/>'
		+ '<iframe src="" name="upload"/></form>');
	form.find('input[name=client_id]').val(sessionId);
	this.$cancel = form.find('input[value=Cancel]').click($.proxy(this,
			'cancel_upload'));
	this.$iframe = form.find('iframe');
	this.$imageInput = form.find('input[name=image]').change(
			$.proxy(this, 'on_image_chosen'));
	this.$toggle = form.find('#toggle').click($.proxy(this, 'on_toggle'));
	if (nashi.upload) {
		this.$imageInput.hide();
		this.$toggle.hide();
	}
	this.spoiler = 0;
	this.nextSpoiler = Math.floor(Math.random() * spoilerCount);
	return form;
};

PF.on_image_chosen = function () {
	if (!this.$imageInput.val()) {
		this.uploadStatus.text('');
		return;
	}
	this.prep_upload();
	var form = this.uploadForm;
	if (!this.num) {
		var alloc = $('<input type="hidden" name="alloc"/>');
		var request = this.make_alloc_request(null);
		form.append(alloc.val(JSON.stringify(request)));
	}
	form.find('input[name=spoiler]').val(this.spoiler);
	form.submit();
	this.$imageInput.attr('disabled', true);
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
				+ mediaURL + 'kana/' + path + '")');
	}
};
