var Post = Backbone.Model.extend({
	idAttribute: 'num',
});

var Replies = Backbone.Collection.extend({model: Post});

var Thread = Backbone.Model.extend({
	idAttribute: 'num',
	initialize: function () {
		if (!this.get('replies'))
			this.set('replies', new Replies([]));
	},
});

var ThreadCollection = Backbone.Collection.extend({
	model: Thread,

	lookup: function (num, op) {
		var thread = this.get(op) || UnknownThread;
		return (num == op) ? thread : thread.get('replies').get(num);
	},
});

var Threads = new ThreadCollection();
var UnknownThread = new Thread();

function model_link(key) {
	return function (event) {
		this.model.set(key, $(event.target).val());
	};
}

var Section = Backbone.View.extend({
	tagName: 'section',

	initialize: function () {
		this.listenTo(this.model, {
			'change:locked': this.renderLocked,
			'change:spoiler': this.renderSpoiler,
			destroy: this.remove,
		});
		this.listenTo(this.model.get('replies'), {
			remove: this.removePost,
		});
	},

	renderLocked: function (model, locked) {
		this.$el.toggleClass('locked', !!locked);
	},

	renderSpoiler: function (model, spoiler) {
		var $img = this.$el.children('figure').find('img');
		var sp = oneeSama.spoiler_info(spoiler, true);
		$img.replaceWith($('<img>', {
			src: sp.thumb, width: sp.dims[0], height: sp.dims[1],
		}));
	},

	remove: function () {
		var replies = this.model.get('replies');
		_.each(replies.models, function (post) {
			clear_post_links(post, replies);
		});
		replies.reset();

		this.$el.next('hr').andSelf().remove();
		this.stopListening();
	},

	removePost: function (model) {
		model.trigger('removeSelf');
	},
});

/* XXX: Move into own views module once more substantial */
var Article = Backbone.View.extend({
	tagName: 'article',
	initialize: function () {
		this.listenTo(this.model, {
			'change:backlinks': this.renderBacklinks,
			'change:editing': this.renderEditing,
			'change:image': this.renderImage,
			'change:spoiler': this.renderSpoiler,
			'removeSelf': this.remove,
		});
	},

	render: function () {
		var html = oneeSama.mono(this.model.attributes);
		this.setElement($($.parseHTML(html)).filter('article')[0]);
		return this;
	},

	renderBacklinks: function () {
		if (options.get('nobacklinks'))
			return this; /* ought to disconnect handler? */
		var backlinks = this.model.get('backlinks');
		var $list = this.$el.find('small');
		if (!backlinks || !backlinks.length) {
			$list.remove();
			return this;
		}
		if (!$list.length)
			$list = $('<small/>', {text: 'Replies:'}).appendTo(
					this.$el);
		// TODO: Sync up DOM gracefully instead of clobbering
		$list.find('a').remove();
		_.each(backlinks, function (num) {
			var $a = $('<a/>', {href: '#'+num, text: '>>'+num});
			$list.append(' ', $a);
		});
		return this;
	},

	renderEditing: function (model, editing) {
		this.$el.toggleClass('editing', !!editing);
		if (!editing)
			this.$('blockquote')[0].normalize();
	},

	renderImage: function (model, image) {
		var hd = this.$('header'), fig = this.$('figure');
		if (!image)
			fig.remove();
		else if (hd.length && !fig.length) {
			/* Is this focus business necessary here? */
			var focus = get_focus();

			insert_image(image, hd, false);

			if (focus)
				focus.focus();
		}
	},

	renderSpoiler: function (model, spoiler) {
		var $img = this.$('figure').find('img');
		var sp = oneeSama.spoiler_info(spoiler, false);
		$img.replaceWith($('<img>', {
			src: sp.thumb,
			width: sp.dims[0], height: sp.dims[1],
		}));
	},
});

/* BATCH DOM UPDATE DEFER */

var deferredChanges = {links: {}, backlinks: {}};
var haveDeferredChanges = false;

/* this runs just before every _outermost_ wrap_dom completion */
Backbone.on('flushDomUpdates', function () {
	if (!haveDeferredChanges)
		return;
	haveDeferredChanges = false;

	for (var attr in deferredChanges) {
		var deferred = deferredChanges[attr];
		var empty = true;
		for (var id in deferred) {
			deferred[id].trigger('change:'+attr);
			empty = false;
		}
		if (!empty)
			deferredChanges[attr] = {};
	}
});

/* LINKS */

function add_post_links(src, links, op) {
	if (!src || !links)
		return;
	var thread = Threads.get(op);
	var srcLinks = src.get('links') || [];
	var repliedToMe = false;
	for (var destId in links) {
		var dest = thread && thread.get('replies').get(destId);
		if (!dest) {
			/* Dest doesn't exist yet; track it anyway */
			dest = new Post({id: destId, shallow: true});
			UnknownThread.get('replies').add(dest);
		}
		if (dest.get('mine'))
			repliedToMe = true;
		var destLinks = dest.get('backlinks') || [];
		/* Update links and backlinks arrays in-order */
		var i = _.sortedIndex(srcLinks, dest.id);
		if (srcLinks[i] == dest.id)
			continue;
		srcLinks.splice(i, 0, dest.id);
		destLinks.splice(_.sortedIndex(destLinks, src.id), 0, src.id);
		force_post_change(src, 'links', srcLinks);
		force_post_change(dest, 'backlinks', destLinks);
	}

	if (repliedToMe && !src.get('mine')) {
		/* Should really be triggered only on the thread */
		Backbone.trigger('repliedToMe');
	}
}

function force_post_change(post, attr, val) {
	if (val === undefined && post.has(attr))
		post.unset(attr);
	else if (post.get(attr) !== val)
		post.set(attr, val);
	else if (!(post.id in deferredChanges[attr])) {
		/* We mutated the existing array, so `change` won't fire.
		   Dumb hack ensues. Should extend Backbone or something. */
		/* Also, here we coalesce multiple changes just in case. */
		/* XXX: holding a direct reference to post is gross */
		deferredChanges[attr][post.id] = post;
		haveDeferredChanges = true;
	}
}

function clear_post_links(post, replies) {
	if (!post)
		return;
	_.each(post.get('links') || [], function (destId) {
		var dest = replies.get(destId);
		if (!dest)
			return;
		var backlinks = dest.get('backlinks') || [];
		var i = backlinks.indexOf(post.id);
		if (i < 0)
			return;
		backlinks.splice(i, 1);
		if (!backlinks.length)
			backlinks = undefined;
		force_post_change(dest, 'backlinks', backlinks);
	});
	_.each(post.get('backlinks') || [], function (srcId) {
		var src = replies.get(srcId);
		if (!src)
			return;
		var links = src.get('links') || [];
		var i = links.indexOf(post.id);
		if (i < 0)
			return;
		links.splice(i, 1);
		if (!links.length)
			links = undefined;
		force_post_change(src, 'links', links);
	});
	post.unset('links', {silent: true});
	post.unset('backlinks');
}

function extract_post_model(el) {
	/* incomplete */
	var info = {num: parseInt(el.id, 10)};
	var $article = $(el);
	/* TODO: do these all in one pass */
	var $header = $article.children('header');
	var $b = $header.find('b');
	if ($b.length)
		info.name = $b.text();
	var $code = $header.find('code');
	if ($code.length)
		info.trip = $code.text();
	var $time = $header.find('time');
	if ($time.length)
		info.time = new Date($time.attr('datetime')).getTime();

	var $fig = $article.children('figure');
	if ($fig.length) {
		var $cap = $fig.children('figcaption');
		var image = {
			MD5: $fig.data('md5'),
			src: $cap.children('a').text(),
		};

		/* guess for now */
		image.thumb = image.src;

		var m = $cap.find('i').text().match(
				/^\(\d+ \w+, (\d+)x(\d+),/);
		if (m)
			image.dims = [parseInt(m[1], 10), parseInt(m[2], 10)];
		info.image = image;
	}
	return new Post(info);
}

function extract_thread_model(section) {
	var replies = [];
	for (var i = 0; i < section.childElementCount; i++) {
		var el = section.children[i];
		if (el.tagName != 'ARTICLE')
			continue;
		var post = extract_post_model(el);
		new Article({model: post, el: el});
		replies.push(post);
	}
	return new Thread({
		num: parseInt(section.id, 10),
		replies: new Replies(replies),
	});
}

(function scan_threads_for_extraction() {
	var bod = document.body;
	var threads = [];
	for (var i = 0; i < bod.childElementCount; i++) {
		var el = bod.children[i];
		if (el.tagName != 'SECTION')
			continue;
		var thread = extract_thread_model(el);
		new Section({model: thread, el: el});
		threads.push(thread);
	}
	Threads.add(threads);
})();
