/*
 * Common methods to both OP and regular posts
 */

var $ = require('jquery'),
	_ = require('underscore'),
	main = require('../main'),
	imager = require('./imager'),
	Menu = require('./menu'),
	common = main.common,
	options = main.options,
	state = main.state;

module.exports = {
	events: {
		'click >figure>figcaption>.imageToggle': 'toggleThumbnailVisibility',
		'click >figure>a': 'imageClicked',
		'click >header>nav>a.quote': 'quotePost',
		'click >header>.control': 'renderMenu'
	},
	initCommon: function(){
		this.$blockquote = this.$el.children('blockquote');
		this.listenTo(this.model, {
			'spoiler': this.renderSpoiler,
			'change:image': this.renderImage,
			updateBody: this.updateBody
		});
		this.listenTo(options, {
			'change:thumbs': this.renderImage,
			'change:spoilers': this.toggleSpoiler,
			'change:autogif': this.toggleAutogif,
			'change:anonymise': this.toggleAnonymisation,
			'change:relativeTime': this.renderTime
		});
		// Automatic image expansion
		this.listenTo(imager.massExpander, 'change:expand',
			function(model, expand) {
				this.toggleImageExpansion(expand);
			}
		);
		if (options.get('relativeTime'))
			this.renderTime(null, true);
		this.fun();
		// Anonymise on post insertion
		if (options.get('anonymise'))
			this.toggleAnonymisation(null, true);
		const links = state.linkerCore.get(this.model.get('num'));
		if (links)
			this.renderBacklinks(null, links);
		this.listenTo(state.linkerCore,
			'change:' + this.model.get('num'),
			this.renderBacklinks
		);
		return this;
	},
	updateBody: function(update) {
		main.oneeSama.dice = update.dice;
		main.oneeSama.links = update.links;
		main.oneeSama.callback = this.inject;
		main.oneeSama.$buffer = this.$blockquote;
		main.oneeSama.state = update.state;
		main.oneeSama.fragment(update.frag);
	},
	// Inject various tags into the blockqoute
	inject: function(frag) {
		var $dest = this.$buffer;
		for (var i = 0; i < this.state[1]; i++)
			$dest = $dest.children('del').last();
		if (this.state[0] == common.S_QUOTE)
			$dest = $dest.children('em').last();
		if (this.strong)
			$dest = $dest.children('strong').last();
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
				out = $(common.flatten(frag).join(''));
			else
				out = common.escape_fragment(frag);
		}
		if (out)
			$dest.append(out);
		return out;
	},
	renderTime: function(model, rtime = options.get('relativeTime')) {
		// TEMP: Remove after extraction is properly defered
		main.oneeSama.rTime = rtime;
		let el = this.el.getElementsByTagName('time')[0];
		if (!this.timeStamp)
			this.timeStamp = main.request('time:fromEl', el).getTime();
		// Create new time element from string
		let newEl = new DOMParser()
			.parseFromString(main.oneeSama.time(this.timeStamp), 'text/xml')
			.firstChild;
		el.parentNode.replaceChild(newEl, el);
		if (this.timer)
			clearTimeout(this.timer);
		if (rtime)
			this.timer = setTimeout(this.renderTime.bind(this), 60000);
	},
	renderBacklinks: function(model, links) {
		// No more backlinks, because posts deleted or something
		if (!links && this.$backlinks)
			return main.command('scroll:follow', () => this.$backlinks.remove());
		if (!this.$backlinks) {
			this.$backlinks = $('<small/>')
				.insertAfter(this.$el.children('blockquote'))
		}
		let html = 'Replies:';
		const thread = state.page.get('thread'),
			notBoard = thread !== 0;
		for (var key in links) {
			if (!links.hasOwnProperty(key))
				continue;
			// points to a different thread from the current
			const diff = links[key] !== thread;
			html += common.parseHTML
				` <a class="history" href="${diff && links[key]}#${key}">
					&gt;&gt;${key}${diff && notBoard && ' →'}
				</a>`;
		}
		main.command('scroll:follow', () => this.$backlinks.html(html));
	},
	renderMenu: function(e) {
		new Menu({
			parent: e.target,
			model: this.model
		});
	},
	// Admin JS injections
	fun: function() {
		// Fun goes here
	},
	// Self-delusion tripfag filter
	toggleAnonymisation: function(model, toggle) {
		var $el = this.$el.find('>header>b');
		const name = this.model.get('name');
		if (toggle)
			$el.text(main.lang.anon);
		// No need to change, if no name
		else if (name)
			$el.text(name);
	},
	quotePost: function(e) {
		e.preventDefault();

		// TODO: Set highlighted post

		/*
		 Make sure the selection both starts and ends in the quoted post's
		 blockquote
		 */
		var sel,
			$post = this.$el,
			gsel = getSelection();
		const num = this.model.get('num');

		function isInside(p) {
			var $el = $(gsel[p] && gsel[p].parentElement);
			return $el.closest('blockquote').length
				&& $el.closest('article, section').is($post);
		}

		if (isInside('baseNode') && isInside('focusNode'))
			sel = gsel.toString();
		main.command('scroll:follow', function() {
			main.command('openPostBox', num);
			main.request('postForm').addReference(num, sel);
		});
	}
};

_.extend(module.exports, imager.Hidamari);
