var syncs = {}, nonces = {}, ownPosts = {};
var readOnly = ['archive'];
var $ceiling = $('hr:first');

var connSM = new FSM('load');
var postSM = new FSM('none');
var sessionId;

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
	var $focus = $(window.getSelection().focusNode);
	if ($focus.is('blockquote'))
		return $focus.find('textarea');
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
	postSM.feed('alloc', msg[0]);
};

dispatcher[INSERT_POST] = function (msg) {
	var num = msg[0];
	msg = msg[1];
	if (!msg.op)
		syncs[num] = 1;
	if (msg.nonce && msg.nonce in nonces) {
		delete nonces[msg.nonce];
		ownPosts[num] = true;
		oneeSama.trigger('insertOwnPost', msg.links, msg);

		/* XXX: Need insert/alloc unification already! */
		if (!CurThread || !postForm || !postForm.post)
			return;
		var post = UnknownThread.get(num);
		if (post) {
			UnknownThread.remove(num);
			post.unset('shallow', {silent: true});
			changedPosts[num] = post;
			queue_post_change_flush();
		}
		else {
			post = new Post({id: num});
		}
		var article = new Article({model: post, id: num,
				el: postForm.post[0]});
		post.view = article;
		CurThread.add(post);
		add_post_links(post, msg.links);

		return;
	}
	msg.num = num;
	msg.editing = true;
	var orig_focus = get_focus();
	oneeSama.links = msg.links;
	var $article, $section, hr, bump = true;
	if (msg.op) {
		$section = $('#' + msg.op);
		if (!$section.length)
			return;
		$article = $(oneeSama.mono(msg));
		shift_replies($section);
		$section.children('blockquote,.omit,form,article[id]:last'
				).last().after($article);
		if (!BUMP || is_sage(msg.email)) {
			bump = false;
		}
		else {
			hr = $section.next();
			$section.detach();
			hr.detach();
		}

		if (CurThread) {
			var post = UnknownThread.get(num);
			if (post) {
				UnknownThread.remove(num);
				post.unset('shallow', {silent: true});
				changedPosts[num] = post;
				queue_post_change_flush();
			}
			else {
				post = new Post({id: num});
			}
			var article = new Article({model: post, id: num,
					el: $article[0]});
			post.view = article;
			CurThread.add(post);
			add_post_links(post, msg.links);
		}
	}
	else {
		$section = $(oneeSama.monomono(msg).join(''));
		hr = $('<hr/>');
		if (!postForm)
			$section.append(make_reply_box());
		if (!BUMP) {
			$section.hide();
			hr.hide();
		}
	}

	oneeSama.trigger('afterInsert', msg.op ? $article : $section);
	if (bump) {
		var fencepost = $('body > aside');
		$section.insertAfter(fencepost.length ? fencepost : $ceiling
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
		PF[msg[0].func].call(postForm, msg[0].arg);
};

dispatcher[INSERT_IMAGE] = function (msg) {
	var focus = get_focus();
	var num = msg[0];
	if (postForm && postForm.num == num)
		return postForm.insert_uploaded(msg[1]);
	var hd = $('#' + num + '>header');
	if (hd.length) {
		insert_image(msg[1], hd, false);
		if (focus)
			focus.focus();
	}
};

dispatcher[UPDATE_POST] = function (msg) {
	var num = msg[0], links = msg[4], extra = msg[5];
	if (CurThread)
		add_post_links(CurThread.get(num), links);
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
			return postSM.feed('done');
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
			postSM.feed('done');
		if (num == op)
			return;
	}
	$('section#' + op).next('hr').andSelf().remove();
};

dispatcher[DELETE_IMAGES] = function (msg, op) {
	_.each(msg, function (num) {
		$('#' + num + '>figure').remove();
	});
};

dispatcher[SPOILER_IMAGES] = function (msg, op) {
	_.each(msg, function (num) {
		var post = $('#' + num);
		var $img = post.children('figure').find('img');
		if ($img.length) {
			var sp = spoiler_info(config.FORCED_SPOILER,
					post.is('section'));
			$img.replaceWith($('<img>').attr('src', sp.thumb
				).width(sp.dims[0]).height(sp.dims[1]));
		}
	});
};

function insert_image(info, header, toppu) {
	var fig = $(flatten(oneeSama.gazou(info, toppu)).join(''));
	if (toppu)
		header.before(fig);
	else
		header.after(fig);
}

var samePage = new RegExp('^(?:' + THREAD + ')?(#\\d+)$');
$DOC.on('click', 'a', function (event) {
	var target = $(event.target);
	var href = target.attr('href');
	if (href && (THREAD || postForm)) {
		var q = href.match(/#q(\d+)/);
		if (q) {
			event.preventDefault();
			with_dom(function () {
				var id = parseInt(q[1], 10);
				open_post_box(id);
				postForm.add_ref(id);
			});
		}
		else if (THREAD) {
			q = href.match(samePage);
			if (q) {
				$('.highlight').removeClass('highlight');
				$(q[1]).addClass('highlight');
			}
		}
	}
});

$DOC.on('click', 'del', function (event) {
	$(event.target).toggleClass('reveal');
});

$DOC.on('click', 'nav input', function (event) {
	location.href = $('link[rel=next]').prop('href');
});

dispatcher[SYNCHRONIZE] = connSM.feeder('sync');
dispatcher[INVALID] = connSM.feeder('invalid');

$(function () {
	$('section').each(function () {
		var s = $(this);
		syncs[s.attr('id')] = parseInt(s.attr('data-sync'));

		/* Insert image omission count (kinda dumb) */
		if (!THREAD) {
			var img = parseInt(s.attr('data-imgs')) -
					s.find('img').length;
			if (img > 0) {
				var stat = s.find('.omit');
				var o = stat.text().match(/(\d*)/)[0];
				stat.text(abbrev_msg(parseInt(o), img));
			}
		}
	});

	var m = window.location.hash.match(/^#q?(\d+)$/);
	if (m)
		$('#' + m[1]).addClass('highlight');

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
});
