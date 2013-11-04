var syncs = {}, ownPosts = {};
var readOnly = ['archive'];

var connSM = new FSM('load');
var postSM = new FSM('none');
var CONN_ID;

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
		else if (/^<\/\w+>$/.test(frag.safe))
			out = '';
	}
	if (out === null) {
		if (_.isArray(frag))
			out = $(flatten(frag).join(''));
		else
			out = escape_fragment(frag);
	}
	if (out)
		dest.append(out);
	return out;
}

// TODO: Unify self-updates with OneeSama; this is redundant
oneeSama.hook('insertOwnPost', function (info) {
	if (!postForm || !info.links)
		return;
	postForm.buffer.find('.nope').each(function () {
		var $a = $(this);
		var text = $a.text();
		var m = text.match(/^>>(\d+)/);
		if (!m)
			return;
		var num = m[1], op = info.links[num];
		if (!op)
			return;
		var realRef = postForm.imouto.post_ref(num, op, false);
		var $ref = $(flatten([realRef]).join(''));
		$a.attr('href', $ref.attr('href')).removeAttr('class');
		var refText = $ref.text();
		if (refText != text)
			$a.text(refText);
	});
});

/* Mobile */
function touchable_spoiler_tag(del) {
	del.html = '<del onclick="void(0)">';
}
oneeSama.hook('spoilerTag', touchable_spoiler_tag);

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
	return {stat: stat, omit: parseInt(m[1], 10),
			img: parseInt(m[2] || 0, 10)};
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
	var omitsBefore = omit;
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
	if (omitsBefore <= THREAD_LAST_N && omit > THREAD_LAST_N) {
		var $expand = section.find('header .act');
		if ($expand.length == 1) {
			var num = extract_num(section);
			var $lastN = $(oneeSama.last_n_html(num));
			$expand.after(' ', $lastN);
		}
	}
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
	var orig_focus = get_focus();
	var num = msg[0];
	msg = msg[1];
	if (!msg.op)
		syncs[num] = 1;
	msg.editing = true;
	msg.num = num;

	var el;
	var isOwn = !!(msg.nonce && msg.nonce in nonces);
	if (isOwn) {
		delete nonces[msg.nonce];
		ownPosts[num] = true;
		oneeSama.trigger('insertOwnPost', msg);
		postSM.feed('alloc', msg);

		if (postForm && postForm.el)
			el = postForm.el;
	}
	delete msg.nonce;

	/* This conflict is really dumb. */
	var links = oneeSama.links = msg.links;
	delete msg.links;

	var model, $section, $hr, bump = true;
	if (msg.op) {
		model = UnknownThread.get('replies').get(num);
		if (model) {
			UnknownThread.get('replies').remove(num);
			model.unset('shallow');
			model.set(msg);
		}
		else
			model = new Post(msg);

		var article = new Article({model: model, id: num, el: el});
		if (!el)
			el = article.render().el;

		var thread = Threads.get(msg.op) || UnknownThread;
		thread.get('replies').add(model);
		add_post_links(model, links, msg.op);

		$section = $('#' + msg.op);
		shift_replies($section);
		$section.children('blockquote,.omit,form,article[id]:last'
				).last().after(el);
		if (!BUMP || is_sage(msg.email)) {
			bump = false;
		}
		else {
			$hr = $section.next();
			$section.detach();
			$hr.detach();
		}
	}
	else {
		model = new Thread(msg);
		Threads.add(model);

		if (!el) {
			$section = $($.parseHTML(oneeSama.monomono(msg
					).join('')));
			el = $section.filter('section')[0];
		}
		else {
			$section = $(el);
		}
		var section = new Section({model: model, id: num, el: el});
		$hr = $('<hr/>');
		if (!postForm)
			$section.append(make_reply_box());
		if (!BUMP) {
			$section.hide();
			$hr.hide();
		}
	}

	if (isOwn)
		model.set('mine', true);
	Backbone.trigger('afterInsert', model, $(el));
	if (bump) {
		var fencepost = $('body > aside');
		$section.insertAfter(fencepost.length ? fencepost : $ceiling
				).after($hr);
		spill_page();
	}
	if (orig_focus)
		orig_focus.focus();
};

dispatcher[MOVE_THREAD] = function (msg, op) {
	msg = msg[0];
	msg.num = op;
	var orig_focus = get_focus();

	var model = new Thread(msg);
	Threads.add(model);

	oneeSama.links = msg.links;
	var $el = $($.parseHTML(oneeSama.monomono(msg).join('')));
	var el = $el.filter('section')[0];

	var section = new Section({model: model, id: op, el: el});
	var $hr = $('<hr/>');
	// No make_reply_box since this is archive-only for now
	if (!BUMP) {
		$el.hide();
		$hr.hide();
	}
	if (msg.replyctr > 0) {
		var omitMsg = abbrev_msg(msg.replyctr, msg.imgctr - 1);
		$('<span class="omit"/>').text(omitMsg).appendTo($el);
	}

	Backbone.trigger('afterInsert', model, $el);
	if (BUMP) {
		var fencepost = $('body > aside');
		$el.insertAfter(fencepost.length ? fencepost : $ceiling
				).after($hr);
		spill_page();
	}
	if (orig_focus)
		orig_focus.focus();
};

dispatcher[IMAGE_STATUS] = function (msg) {
	if (postForm)
		ComposerView.prototype[msg[0].func].call(postForm, msg[0].arg);
};

dispatcher[INSERT_IMAGE] = function (msg, op) {
	var focus = get_focus();
	var num = msg[0];
	var thread = Threads.get(op) || UnknownThread;
	var post = thread.get('replies').get(num);

	if (saku && saku.get('num') == num) {
		if (post)
			post.set('image', msg[1], {silent: true}); // TEMP
		postForm.insert_uploaded(msg[1]);
	}
	else if (post)
		post.set('image', msg[1]);

	if (num == MILLION) {
		var $el = $('#' + num);
		$el.css('background-image', oneeSama.gravitas_style(msg[1]));
		var bg = $el.css('background-color');
		$el.css('background-color', 'black');
		setTimeout(function () { $el.css('background-color', bg); }, 500);
	}

	if (focus)
		focus.focus();
};

dispatcher[UPDATE_POST] = function (msg, op) {
	var num = msg[0], links = msg[4], extra = msg[5];
	var state = [msg[2] || 0, msg[3] || 0];
	var thread = Threads.get(op) || UnknownThread;
	var post = op == num ? thread : thread.get('replies').get(num);
	if (post) {
		add_post_links(post, links, op);
		var body = post.get('body') || '';
		post.set({body: body + msg[1], state: state});
	}

	if (num in ownPosts) {
		if (extra)
			extra.links = links;
		else
			extra = {links: links};
		oneeSama.trigger('insertOwnPost', extra);
		return;
	}
	var bq = $('#' + num + ' > blockquote');
	if (bq.length) {
		oneeSama.dice = extra && extra.dice;
		oneeSama.links = links || {};
		oneeSama.callback = inject;
		oneeSama.buffer = bq;
		oneeSama.state = state;
		oneeSama.fragment(msg[1]);
	}
};

dispatcher[FINISH_POST] = function (msg, op) {
	var num = msg[0];
	delete ownPosts[num];
	var thread = Threads.get(op);
	var post;
	if (op == num) {
		if (!thread)
			return;
		post = thread;
	}
	else {
		if (!thread)
			thread = UnknownThread;
		post = thread.get('replies').get(num);
	}

	if (post)
		post.set('editing', false);
};

dispatcher[DELETE_POSTS] = function (msg, op) {
	var replies = (Threads.get(op) || UnknownThread).get('replies');
	var $section = $('#' + op);
	var ownNum = saku && saku.get('num');
	_.each(msg, function (num) {
		var postVisible = $('#' + num).is('article');
		delete ownPosts[num];
		var post = replies.get(num);
		clear_post_links(post, replies);
		if (num === ownNum)
			return postSM.feed('done');
		if (num == lockTarget)
			set_lock_target(null);
		if (post)
			replies.remove(post);

		if (!THREAD && !postVisible) {
			/* post not visible; decrease omit count */
			var info = section_abbrev($section);
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
	delete ownPosts[op];
	if (saku) {
		var num = saku.get('num');
		if ((saku.get('op') || num) == op)
			postSM.feed('done');
		if (num == op)
			return;
	}
	var thread = Threads.get(op);
	if (thread)
		thread.trigger('destroy', thread, thread.collection);
};

dispatcher[LOCK_THREAD] = function (msg, op) {
	var thread = Threads.get(op);
	if (thread)
		thread.set('locked', true);
};

dispatcher[UNLOCK_THREAD] = function (msg, op) {
	var thread = Threads.get(op);
	if (thread)
		thread.set('locked', false);
};

dispatcher[DELETE_IMAGES] = function (msg, op) {
	var replies = (Threads.get(op) || UnknownThread).get('replies');
	_.each(msg, function (num) {
		var post = replies.get(num);
		if (post)
			post.unset('image');
	});
};

dispatcher[SPOILER_IMAGES] = function (msg, op) {
	var replies = (Threads.get(op) || UnknownThread).get('replies');
	_.each(msg, function (info) {
		var num = info[0];
		var post = num == op ? Threads.get(num) : replies.get(num);
		if (post)
			post.set('spoiler', info[1]);
	});
};

function insert_image(info, header, toppu) {
	var fig = $(flatten(oneeSama.gazou(info, toppu)).join(''));
	if (toppu)
		header.before(fig);
	else
		header.after(fig);
}

function set_highlighted_post(num) {
	$('.highlight').removeClass('highlight');
	$('article#' + num).addClass('highlight');
}

var samePage = new RegExp('^(?:' + THREAD + ')?#(\\d+)$');
$DOC.on('click', 'a', function (event) {
	var target = $(this);
	var href = target.attr('href');
	if (href && (THREAD || postForm)) {
		var q = href.match(/#q(\d+)/);
		if (q) {
			event.preventDefault();
			var id = parseInt(q[1], 10);
			set_highlighted_post(id);
			with_dom(function () {
				open_post_box(id);
				postForm.add_ref(id);
			});
		}
		else if (THREAD) {
			q = href.match(samePage);
			if (q)
				set_highlighted_post(q[1]);
		}
	}
});

$DOC.on('click', 'del', function (event) {
	if (!event.spoilt) {
		event.spoilt = true;
		$(event.target).toggleClass('reveal');
	}
});

$DOC.on('click', '.pagination input', function (event) {
	location.href = $('link[rel=next]').prop('href');
});

dispatcher[SYNCHRONIZE] = connSM.feeder('sync');
dispatcher[INVALID] = connSM.feeder('invalid');

function lookup_model_path(path) {
	var o = window;
	if (!_.isArray(path))
		return o[path];
	o = o[path[0]];
	if (o) {
		for (var i = 1; i < path.length; i++) {
			o = o.get(path[i]);
			if (!o)
				break;
		}
	}
	return o;
}

dispatcher[MODEL_SET] = function (msg, op) {
	var target = lookup_model_path(msg[0]);
	if (target && target.set)
		target.set(msg[1]);
};

dispatcher[COLLECTION_RESET] = function (msg, op) {
	var target = lookup_model_path(msg[0]);
	if (target && target.reset)
		target.reset(msg[1]);
};

dispatcher[COLLECTION_ADD] = function (msg, op) {
	var target = lookup_model_path(msg[0]);
	if (target && target.add)
		target.add(msg[1], {merge: true});
};

(function () {
	var m = window.location.hash.match(/^#q?(\d+)$/);
	if (m)
		set_highlighted_post(m[1]);

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

	$('time').each(function () {
		var t = $(this);
		var d = t.attr('datetime').replace(/-/g, '/'
			).replace('T', ' ').replace('Z', ' GMT');
		t.html(readable_time(new Date(d).getTime()));
	});

	$('del').attr('onclick', 'void(0)');
})();
