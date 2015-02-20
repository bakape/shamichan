function inject(frag) {
	var dest = this.buffer;
	for (var i = 0; i < this.state[1]; i++)
		dest = dest.children('del:last');
	if (this.state[0] == DEF.S_QUOTE)
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
		if (Array.isArray(frag))
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
	if (rem < hotConfig.ABBREVIATED_REPLIES)
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
		if (rem-- < hotConfig.ABBREVIATED_REPLIES)
			break;
		if (cull.has('figure').length)
			img++;
		omit++;
		// Trigger the remove even on the appropriate post model
		Posts.findWhere({num: extract_num(cull)}).trigger('removeSelf');
	}
	$stat.text(abbrev_msg(omit, img));
}

function spill_page() {
	if (THREAD)
		return;
	/* Ugh, this could be smarter. */
	var ss = $('body > section[id]:visible');
	for (var i = hotConfig.THREADS_PER_PAGE; i < ss.length; i++)
		$(ss[i]).prev('hr').andSelf().hide();

}

var dispatcher = {};

dispatcher[DEF.INSERT_POST] = function (msg) {
	var orig_focus = get_focus();
	var num = msg[0];
	msg = msg[1];
	var isThread = !msg.op;
	if (isThread)
		syncs[num] = 1;
	msg.editing = true;
	msg.num = num;

	// did I create this post?
	var el;
	var nonce = msg.nonce;
	delete msg.nonce;
	var myNonce = get_nonces()[nonce];
	var bump = BUMP;
	var myTab = myNonce && myNonce.tab == TAB_ID;
	if (myTab) {
		// posted in this tab; transform placeholder
		ownPosts[num] = true;
		oneeSama.trigger('insertOwnPost', msg);
		postSM.feed('alloc', msg);
		bump = false;
		// delete only after a delay so all tabs notice that it's ours
		setTimeout(destroy_nonce.bind(null, nonce), 10*1000);
		// if we've already made a placeholder for this post, use it
		if (postForm && postForm.el)
			el = postForm.el;
	}

	/* This conflict is really dumb. */
	var links = oneeSama.links = msg.links;
	delete msg.links;

	// create model or fill existing shallow model
	var model;
	if (!isThread) {
		model = UnknownThread.get('replies').get(num);
		if (model) {
			UnknownThread.get('replies').remove(num);
			model.unset('shallow');
			model.set(msg);
		}
		else
			model = new Post(msg);
	}
	else {
		model = new Thread(msg);
	}

	if (myNonce) {
		model.set('mine', true);
		Mine.write(num, Mine.now());
	}

	// insert it into the DOM
	var $section, $hr;
	if (!isThread) {
		var article = new Article({model: model, id: num, el: el});
		if (!el)
			el = article.render().el;

		var thread = Threads.lookup(msg.op, msg.op);
		thread.get('replies').add(model);
		add_post_links(model, links, msg.op);

		$section = $('#' + msg.op);
		shift_replies($section);
		$section.children('blockquote,.omit,form,article[id]:last'
				).last().after(el);
		if (is_sage(msg.email)) {
			bump = false;
		}
		if (postForm) {
			// don't bump due to replies while posting (!)
			bump = false;
		}
		else
			article.autoExpandImage();
		if (bump) {
			$hr = $section.next();
			$section.detach();
			$hr.detach();
		}
	}
	else {
		Threads.add(model);
	}

	// only add new threads on /live
	if (isThread && BUMP) {
		if (!el) {
			$section = $($.parseHTML(oneeSama.monomono(msg
					).join('')));
			$section = $section.filter('section');
			el = $section[0];
		}
		else {
			$section = $(el);
		}
		var section = new Section({model: model, id: num, el: el});
		$hr = $('<hr class="sectionHr"/>');
		if (!postForm)
			$section.append(make_reply_box());
	}

	// Add to all post collection
	Posts.add(model);
	Backbone.trigger('afterInsert', model, $(el));
	if (bump) {
		var fencepost = $('body > aside');
		$section.insertAfter(fencepost.length ? fencepost : $ceiling);
		if ($hr)
			$section.after($hr);
		spill_page();
	}
	if (orig_focus)
		orig_focus.focus();
};

dispatcher[DEF.MOVE_THREAD] = function (msg, op) {
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

dispatcher[DEF.INSERT_IMAGE] = function (msg, op) {
	var focus = get_focus();
	var num = msg[0];
	var post = Threads.lookup(num, op);

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

dispatcher[DEF.UPDATE_POST] = function (msg, op) {
	var num = msg[0], links = msg[4], extra = msg[5];
	var state = [msg[2] || 0, msg[3] || 0];
	var post = Threads.lookup(num, op);
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

dispatcher[DEF.FINISH_POST] = function (msg, op) {
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

dispatcher[DEF.DELETE_POSTS] = function (msg, op) {
	var replies = Threads.lookup(op, op).get('replies');
	var $section = $('#' + op);
	var ownNum = saku && saku.get('num');
	msg.forEach(function (num) {
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

dispatcher[DEF.DELETE_THREAD] = function (msg, op) {
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

dispatcher[DEF.LOCK_THREAD] = function (msg, op) {
	var thread = Threads.get(op);
	if (thread)
		thread.set('locked', true);
};

dispatcher[DEF.UNLOCK_THREAD] = function (msg, op) {
	var thread = Threads.get(op);
	if (thread)
		thread.set('locked', false);
};

dispatcher[DEF.DELETE_IMAGES] = function (msg, op) {
	var replies = Threads.lookup(op, op).get('replies');
	msg.forEach(function (num) {
		var post = replies.get(num);
		if (post)
			post.unset('image');
	});
};

dispatcher[DEF.SPOILER_IMAGES] = function (msg, op) {
	var thread = Threads.get(op);
	var replies = thread.get('replies');
	msg.forEach(function (info) {
		var num = info[0];
		var post = (num == op) ? thread : replies.get(num);
		if (post)
			post.trigger('spoiler', info[1]);
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
	if (href) {
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

dispatcher[DEF.SYNCHRONIZE] = connSM.feeder('sync');
dispatcher[DEF.INVALID] = connSM.feeder('invalid');

dispatcher[DEF.ONLINE_COUNT] = function(msg){
	$('#onlineCount').text('['+msg[0]+']');
};

dispatcher[DEF.HOT_INJECTION] = function(msg){
	// Request new varibles, if hashes don't match
	if (msg[0] == false && msg[1] != configHash)
		send([DEF.HOT_INJECTION, true]);
	// Update variables and hash
	else if (msg[0] == true){
		configHash = msg[1];
		config = msg[2][0];
		imagerConfig = msg[2][1];
		reportConfig = msg[2][2];
		hotConfig = msg[2][3];
	}
};

function lookup_model_path(path) {
	var o = window;
	if (!Array.isArray(path))
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

dispatcher[DEF.MODEL_SET] = function (msg, op) {
	var target = lookup_model_path(msg[0]);
	if (target && target.set)
		target.set(msg[1]);
};

dispatcher[DEF.COLLECTION_RESET] = function (msg, op) {
	var target = lookup_model_path(msg[0]);
	if (target && target.reset)
		target.reset(msg[1]);
};

dispatcher[DEF.COLLECTION_ADD] = function (msg, op) {
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

	$('del').attr('onclick', 'void(0)');
})();
