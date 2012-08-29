var Post = Backbone.Model.extend({});

var Thread = Backbone.Collection.extend({model: Post});

/* TODO: Multiplex */
var CurThread;
var UnknownThread = new Thread([]);

function lookup_post(id) {
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
		this.model.on('change:backlinks', this.renderBacklinks, this);
	},
	renderBacklinks: function () {
		if (!options.backlinks)
			return this; /* ought to disconnect handler? */
		var backlinks = this.model.get('backlinks');
		var $list = this.$el.find('small');
		if (!backlinks || !backlinks.length) {
			$list.remove();
			return this;
		}
		if (!$list.length)
			$list = $('<small>Replies:</small>').appendTo(
					this.$el);
		// TODO: Sync up DOM gracefully instead of clobbering
		$list.find('a').remove();
		_.each(backlinks, function (num) {
			$list.append(' <a href="#'+num+'">&gt;&gt;'+num+'</a>');
		});
		return this;
	},
});

/* BATCH DOM UPDATE DEFER */

var changedPosts = {}, changeFlushTimeout = 0;

function queue_post_change_flush() {
	if (!changeFlushTimeout)
		changeFlushTimeout = setTimeout(flush_wrapped, 0);
}

function flush_post_changes() {
	if (changeFlushTimeout) {
		clearTimeout(changeFlushTimeout);
		changeFlushTimeout = 0;
	}
	for (var id in changedPosts)
		changedPosts[id].change();
	changedPosts = {};
}

var flush_wrapped = _.wrap(flush_post_changes, with_dom);

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
	/* XXX: We mutated the existing array, so `change` won't fire.
	   Dumb hack ensues. Should extend Backbone or something. */
	var opts = {silent: true};
	post.set(attr, null, opts);
	if (val !== undefined)
		post.set(attr, val, opts);
	else
		post.unset(attr, opts);
	/* Defer `change` kickoffs */
	if (!(post.id in changedPosts))
		changedPosts[post.id] = post;
	queue_post_change_flush();
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
	post.unset('links', {silent: true});
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
	post.unset('backlinks', {silent: true});
	changedPosts[post.id] = post;
}

(function () {
	if (!THREAD)
		return;
	var replies = [];
	$('article').each(function () {
		var id = extract_num($(this));
		var post = new Post({id: id});
		var article = new Article({model: post, id: id, el: this});
		post.view = article; // bleh
		replies.push(post);
	});
	CurThread = new Thread(replies);
})();
