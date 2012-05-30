var Post = Backbone.Model.extend({});

var Thread = Backbone.Collection.extend({model: Post});

/* TODO: Multiplex */
var CurThread;

/* XXX: Move into own views module once more substantial */
var Article = Backbone.View.extend({
	tagName: 'article',
	initialize: function () {
		this.model.on('change:backlinks', this.renderBacklinks, this);
	},
	renderBacklinks: function () {
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
		changeFlushTimeout = setTimeout(flush_post_changes, 0);
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

/* LINKS */

function add_post_links(src, links) {
	if (!src || !links)
		return;
	for (var destId in links) {
		var dest = CurThread.get(destId);
		if (!dest) {
			/* TODO: Also track remote posts (shallowly) */
			continue;
		}
		var srcLinks = src.get('links') || [];
		var destLinks = dest.get('backlinks') || [];
		/* Update links and backlinks arrays in-order */
		var i = _.sortedIndex(srcLinks, dest.id);
		if (srcLinks[i] == dest.id)
			continue;
		srcLinks.splice(i, 0, dest.id);
		destLinks.splice(_.sortedIndex(destLinks, src.id), 0, src.id);
		/* XXX: We mutated the existing array, so `change` won't fire.
		   Dumb hack ensues. Should extend Backbone or something. */
		var opts = {silent: true};
		src.set('links', null, opts);
		src.set('links', srcLinks, opts);
		dest.set('backlinks', null, opts);
		dest.set('backlinks', destLinks, opts);
		/* Defer `change` kickoffs */
		if (!(src.id in changedPosts))
			changedPosts[src.id] = src;
		if (!(dest.id in changedPosts))
			changedPosts[dest.id] = dest;
		queue_post_change_flush();
	}
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
