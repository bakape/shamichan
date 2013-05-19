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

var ThreadCollection = Backbone.Collection.extend({model: Thread});

var Threads = new ThreadCollection();
var UnknownThread = new Thread();

function lookup_post(id) {
	var thread = Threads.get(THREAD);
	if (!id || !thread)
		return null;
	return thread.get('replies').get(id) ||
			UnknownThread.get('replies').get(id);
}

function model_link(key) {
	return function (event) {
		this.model.set(key, $(event.target).val());
	};
}

var Section = Backbone.View.extend({
	tagName: 'section',

	initialize: function () {
		this.listenTo(this.model, {
			destroy: this.remove,
		});
	},

	remove: function () {
		var replies = this.model.get('replies');
		_.each(replies.models, clear_post_links);
		replies.reset();

		this.$el.next('hr').andSelf().remove();
		this.stopListening();
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
			'destroy': this.remove,
		});
	},

	renderBacklinks: function () {
		if (options.nobacklinks)
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

function add_post_links(src, links) {
	if (!src || !links)
		return;
	var srcLinks = src.get('links') || [];
	for (var destId in links) {
		var dest = lookup_post(destId);
		if (!dest) {
			/* Dest doesn't exist yet; track it anyway */
			dest = new Post({id: destId, shallow: true});
			UnknownThread.get('replies').add(dest);
		}
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

function clear_post_links(post) {
	if (!post)
		return;
	_.each(post.get('links') || [], function (destId) {
		var dest = lookup_post(destId);
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
		var src = lookup_post(srcId);
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

function extract_post_model($article) {
	/* incomplete */
	var info = {num: extract_num($article)};
	var $header = $article.children('header');
	var $b = $header.find('b');
	if ($b)
		info.name = $b.text();
	var $time = $header.find('time');
	if ($time.length)
		info.time = new Date($time.attr('datetime')).getTime();

	var $fig = $article.children('figure');
	if ($fig.length) {
		var $cap = $fig.children('figcaption');
		var image = {
			MD5: $fig.data('md5'),
			src: $cap.find('a').text(),
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

function extract_thread_model($section) {
	var replies = [];
	$section.find('article').each(function () {
		var post = extract_post_model($(this));
		new Article({model: post, id: post.id, el: this});
		replies.push(post);
	});
	return new Thread({
		num: extract_num($section),
		replies: new Replies(replies),
	});
}

$('section').each(function () {
	var $section = $(this);
	var thread = extract_thread_model($section);
	new Section({model: thread, el: this});
	Threads.add(thread);
});
