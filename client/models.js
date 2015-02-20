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

// All posts currently displayed
var Posts = new Backbone.Collection();

var Threads = new ThreadCollection();
var UnknownThread = new Thread();

function model_link(key) {
	return function (event) {
		this.model.set(key, $(event.target).val());
	};
}

// Keeps threads non-laggy by keeping displayed post count within lastN
function unloadTopPost(){
	var m = location.search.match(/last=(\d+)/);
	if (!m || $(Mouseover.get('target')).is('a, img, video') || CurThread.get('replies').length <= parseInt(m[1], 10)+5)
		return;
	CurThread.get('replies').shift().trigger('removeSelf');
	var $omit = $('.omit');
	if (!$omit.length){
		$omit = $('\t<span/>', {'class': 'omit'}).text(abbrev_msg(1));
		var url = THREAD;
		if (!!location.hash)
			url += location.hash;
		$omit.append(action_link_html(url, 'See all')+'\n');
		$('section>blockquote').after($omit);
	}
	else {
		var m = $omit.html().match(/^(\d+)(.*)/);
		$('.omit').html(parseInt(m[1])+1+m[2]);
	}
	unloadTopPost();
}

var PostMixins = {
	commonListeners: function(){
		this.listenTo(options, {
			'change:thumbs': this.changeThumbnailStyle,
			'change:noSpoilers': this.toggleSpoiler,
			'change:autogif': this.toggleAutogif,
		});
		this.listenTo(massExpander, {
			'change:expand': this.toggleImageExpansion,
		});
	},

	renderRelativeTime: function(){
		if (oneeSama.rTime){
			var $time = this.$el.find('time').first();
			var t = date_from_time_el($time[0]).getTime();
			var timer = setInterval(function(){
				$time.html(oneeSama.relative_time(t, new Date().getTime()));
			}, 60000);
			this.listenToOnce(this.model, 'removeSelf', function(){
				clearInterval(timer);
			});
		}
	},
};

var Section = Backbone.View.extend({
	tagName: 'section',

	initialize: function () {
		this.listenTo(this.model, {
			'change:hide': this.renderHide,
			'change:locked': this.renderLocked,
			'spoiler': this.renderSpoiler,
			destroy: this.remove,
		});
		// Sections get added to multiple collections as both thread OPs and
		// generic posts. Prevent duplication by calling only once
		this.listenToOnce(this.model, {
			'add': this.renderRelativeTime
		});
		this.listenTo(this.model.get('replies'), {
			remove: this.removePost,
		});
		this.commonListeners();
	},

	renderHide: function (model, hide) {
		this.$el.next('hr.sectionHr').andSelf().toggle(!hide);
	},

	renderLocked: function (model, locked) {
		this.$el.toggleClass('locked', !!locked);
	},

	remove: function () {
		var replies = this.model.get('replies');
		replies.each(function (post) {
			clear_post_links(post, replies);
		});
		replies.reset();

		this.$el.next('hr.sectionHr').andSelf().remove();
		// Remove from all Posts collection
		Posts.remove(this.model);
		this.stopListening();
	},

	removePost: function (model) {
		model.trigger('removeSelf');
	},
});
_.extend(Section.prototype, Hidamari, PostMixins);

/* XXX: Move into own views module once more substantial */
var Article = Backbone.View.extend({
	tagName: 'article',
	initialize: function () {
		this.listenTo(this.model, {
			'change:backlinks': this.renderBacklinks,
			'change:editing': this.renderEditing,
			'change:hide': this.renderHide,
			'change:image': this.renderImage,
			'spoiler': this.renderSpoiler,
			'removeSelf': this.bumplessRemove,
			'add': function(){
				this.renderRelativeTime();
				this.fun();
			},
		});
		this.commonListeners();
		if (options.get('postUnloading') && CurThread)
			this.listenTo(this.model, {
				'add': unloadTopPost
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
		backlinks.forEach(function (num) {
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

	renderHide: function (model, hide) {
		this.$el.toggle(!hide);
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
			this.autoExpandImage();
		}
	},

	// To not shift the scroll position on remove
	bumplessRemove: function(){
		var pos = $(window).scrollTop();
		if (!at_bottom() && this.$el.offset().top < pos)
			// Not sure why we need the extra 2 pixels, but we do
			$(window).scrollTop(pos - this.$el.outerHeight() - 2);
		Posts.remove(this.model);
		this.remove();
	},

	fun: function(){
		// Fun goes here
	},
});
_.extend(Article.prototype, Hidamari, PostMixins);

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

// Centralised mouseover target tracking
var Mouseover = new Backbone.Model({target: null});

if (!isMobile) {
	$DOC.on('mouseover', function(e) {
		Mouseover.set('target', e.target);
	});
}

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
		Backbone.trigger('repliedToMe',src);
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
	(post.get('links') || []).forEach(function (destId) {
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
	(post.get('backlinks') || []).forEach(function (srcId) {
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


