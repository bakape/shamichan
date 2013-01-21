var Post = Backbone.Model.extend({});

var Thread = Backbone.Collection.extend({model: Post});

/* TODO: Multiplex */
var CurThread;
var UnknownThread = new Thread([]);

function lookup_post(id) {
	if (!CurThread || !id)
		return null;
	return CurThread.get(id) || UnknownThread.get(id);
}

function model_link(key) {
	return function (event) {
		this.model.set(key, $(event.target).val());
	};
}

/* XXX: Move into own views module once more substantial */
var Article = Backbone.View.extend({
	tagName: 'article',
	initialize: function () {
		this.listenTo(this.model, 'change:backlinks',
				this.renderBacklinks);
		this.listenTo(this.model, 'change:editing',
				this.renderEditing);
		this.listenTo(this.model, 'change:image',
				this.renderImage);
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

/* this runs after EVERY outermost wrap_dom completion */
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
	for (var destId in links) {
		var dest = lookup_post(destId);
		if (!dest) {
			/* Dest doesn't exist yet; track it anyway */
			dest = new Post({id: destId, shallow: true});
			UnknownThread.add(dest);
		}
		var srcLinks = src.get('links') || [];
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

function extract_model_info($article) {
	/* incomplete */
	var info = {};
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
	return info;
}

(function () {
	if (!THREAD)
		return;
	var replies = [];
	$('article').each(function () {
		var $article = $(this);
		var info = extract_model_info($article);
		var id = extract_num($article);
		info.id = id;
		var post = new Post(info);
		var article = new Article({model: post, id: id, el: this});
		post.view = article; // bleh
		replies.push(post);
	});
	CurThread = new Thread(replies);
})();
