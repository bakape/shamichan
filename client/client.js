var BOARD, THREAD, BUMP, PAGE;
var syncs = {}, nonces = {}, ownPosts = {};
var readOnly = ['archive'];
var $ceiling, $sizer;
var lockedToBottom, lockKeyHeight;
var postForm, preview, previewNum;
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
oneeSama.hook('insertOwnPost', function (links, info) {
	if (!postForm || !links)
		return links;
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
	return links;
});

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

dispatcher[ALLOCATE_POST] = function (msg) {
	if (postForm)
		postForm.on_allocation(msg[0]);
	else
		send([FINISH_POST]); // Huh. Just tidy up.
};

dispatcher[INSERT_POST] = function (msg) {
	var num = msg[0];
	msg = msg[1];
	if (msg.nonce && msg.nonce in nonces) {
		delete nonces[msg.nonce];
		ownPosts[num] = true;
		oneeSama.trigger('insertOwnPost', msg.links, msg);
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

dispatcher[MOVE_THREAD] = function (msg, op) {
	msg = msg[0];
	msg.num = op;
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
	if (postForm)
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
	if (num in ownPosts) {
		oneeSama.trigger('insertOwnPost', links, extra);
		return;
	}
	var bq = $('#' + num + '>blockquote');
	if (bq.length) {
		oneeSama.dice = extra && extra.dice;
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

	if (options.inline && target.is('img') && !target.data('skipExpand')) {
		toggle_expansion(target, event);
	}
	else if (target.is('cite')) {
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
	}
	else if (target.is('del')) {
		target.toggleClass('reveal');
	}
}

function toggle_expansion(img, event) {
	event.preventDefault();
	var href = img.parent().attr('href');
	if (href.match(/^\.\.\/outbound\//))
		return;
	var thumb = img.data('thumbSrc');

	with_dom(function () {
		if (thumb) {
			// try to keep the thumbnail in-window for large images
			var h = img.height();
			var th = parseInt(img.data('thumbHeight'), 10);
			var y = img.offset().top, t = $(window).scrollTop();
			if (y < t && th < h)
				window.scrollBy(0, Math.max(th - h,
						y - t - event.clientY + th/2));

			img.replaceWith($('<img>')
				.width(img.data('thumbWidth'))
				.height(th)
				.attr('src', thumb));
			return;
		}
		var caption = img.parent().prev().text();
		var dims = caption.match(/(\d+)x(\d+)/);
		var w = parseInt(dims[1], 10), h = parseInt(dims[2], 10),
			r = window.devicePixelRatio;
		if (r && r > 1) {
			w /= r;
			h /= r;
		}
		img.replaceWith($('<img>').data({
			thumbWidth: img.width(),
			thumbHeight: img.height(),
			thumbSrc: img.attr('src')
		}).attr('src', href).width(w).height(h));
	});
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
	$('#lock').css({visibility: lock ? 'visible' : 'hidden'});
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

function drop_shita(e) {
	e.stopPropagation();
	e.preventDefault();
	var files = e.dataTransfer.files;
	if (!files.length)
		return;
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
	if (files.length > 1) {
		postForm.upload_error('Too many files.');
		return;
	}

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

$(function () {
	$('section').each(function () {
		var s = $(this);
		syncs[s.attr('id')] = parseInt(s.attr('data-sync'));
	});

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
				).css({visibility: 'hidden'}).appendTo('body');
		$(document).scroll(scroll_shita);
		scroll_shita();
	}

	$ceiling = $('hr:first');
	$sizer = $('<pre></pre>');
	$(document.body).append($sizer);
});
