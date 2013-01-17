var syncs = {}, ownPosts = {};
var readOnly = ['archive'];

var connSM = new FSM('load');
var postSM = new FSM('none');
var sessionId;

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
			var $lastN = $(last_n_html(extract_num(section)));
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

/* stupid `links` conflict */
var modelSafeKeys = ['op', 'name', 'trip', 'image', 'time'];

dispatcher[INSERT_POST] = function (msg) {
	var num = msg[0];
	msg = msg[1];
	if (!msg.op)
		syncs[num] = 1;
	if (msg.nonce && msg.nonce in nonces) {
		delete nonces[msg.nonce];
		ownPosts[num] = true;
		oneeSama.trigger('insertOwnPost', msg);
		msg.num = num;
		postSM.feed('alloc', msg);
		delete msg.num;

		if (!CurThread || !postForm || !postForm.el)
			return;
		/* Unify with code below once we have a fuller model */
		var post = UnknownThread.get(num);
		if (post) {
			UnknownThread.remove(num);
			post.unset('shallow');
		}
		else
			post = new Post({id: num});

		_.forEach(modelSafeKeys, function (k) {
			if (msg[k])
				post.set(k, msg[k]);
		});

		var article = new Article({model: post, id: num,
				el: postForm.el});
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
		$article = $($.parseHTML(oneeSama.mono(msg)));
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
				post.unset('shallow');
			}
			else
				post = new Post({id: num});

			_.forEach(modelSafeKeys, function (k) {
				if (msg[k])
					post.set(k, msg[k]);
			});

			var article = new Article({model: post, id: num,
					el: $article.filter('article')[0]});
			post.view = article;
			CurThread.add(post);
			add_post_links(post, msg.links);
		}
	}
	else {
		$section = $($.parseHTML(oneeSama.monomono(msg).join('')));
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

	var section = $($.parseHTML(oneeSama.monomono(msg).join('')));
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
		ComposerView.prototype[msg[0].func].call(postForm, msg[0].arg);
};

dispatcher[INSERT_IMAGE] = function (msg) {
	var focus = get_focus();
	var num = msg[0];
	var post = lookup_post(num);
	if (post)
		post.set('image', msg[1]);

	if (saku && saku.get('num') == num)
		return postForm.insert_uploaded(msg[1]);
	var hd = $('#' + num + ' > header');
	if (hd.length) {
		insert_image(msg[1], hd, false);
		if (focus)
			focus.focus();
	}
};

dispatcher[UPDATE_POST] = function (msg) {
	var num = msg[0], links = msg[4], extra = msg[5];
	if (CurThread)
		add_post_links(lookup_post(num), links);
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
		oneeSama.state = [msg[2] || 0, msg[3] || 0];
		oneeSama.fragment(msg[1]);
	}
};

dispatcher[FINISH_POST] = function (msg) {
	var num = msg[0];
	var post = $('#' + num);
	if (post.length) {
		post.removeClass('editing');
		post[0].normalize();
	}
	delete ownPosts[num];
};

dispatcher[DELETE_POSTS] = function (msg, op) {
	var ownNum = postForm && postForm.num;
	_.each(msg, function (num) {
		delete ownPosts[num];
		if (CurThread)
			clear_post_links(lookup_post(num));
		if (num === ownNum)
			return postSM.feed('done');
		if (num == lockTarget)
			set_lock_target(null);
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
	delete ownPosts[op];
	if (postForm) {
		var num = postForm.num;
		if ((postForm.op || num) == op)
			postSM.feed('done');
		if (num == op)
			return;
	}
	if (CurThread && op == THREAD) {
		_.each(CurThread.models, clear_post_links);
		CurThread.reset();
	}
	$('section#' + op).next('hr').andSelf().remove();
};

dispatcher[LOCK_THREAD] = function (msg, op) {
	$('#' + op).addClass('locked');
};

dispatcher[UNLOCK_THREAD] = function (msg, op) {
	$('#' + op).removeClass('locked');
};

dispatcher[DELETE_IMAGES] = function (msg, op) {
	_.each(msg, function (num) {
		$('#' + num + ' > figure').remove();
	});
};

dispatcher[SPOILER_IMAGES] = function (msg, op) {
	_.each(msg, function (info) {
		var post = $('#' + info[0]);
		var $img = post.children('figure').find('img');
		if ($img.length) {
			var sp = spoiler_info(info[1], post.is('section'));
			$img.replaceWith($('<img>', {
				src: sp.thumb,
				width: sp.dims[0], height: sp.dims[1],
			}));
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
		target.add(msg[1]);
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
		t.text(readable_time(new Date(d).getTime()));
	});

	$('del').attr('onclick', 'void(0)');
})();
