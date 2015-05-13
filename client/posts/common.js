/*
 * Common methods to both OP and regular posts
 */
'use strict';

var $ = require('jquery'),
	common = require('../../common'),
	imager = require('./imager'),
	main = require('../main'),
	options = require('../options'),
	state = require('../state'),
	time = require('../time');

module.exports = {
	events: {
		'click >figure>figcaption>.imageToggle': 'toggleThumbnailVisibility',
		'click >figure>a': 'imageClicked',
		'click >header>nav>a.quote': 'quotePost'
	},

	initCommon: function(){
		this.$blockquote = this.$el.children('blockquote');
		this.listenTo(this.model, {
			'change:hide': this.renderHide,
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
		if (!this.$time) {
			this.$time = this.$el.find('time').first();
			this.time = time.date_from_time_el(this.$time[0]).getTime();
		}
		if (this.hasRelativeTime)
			this.$time.html(main.oneeSama.time(this.time));
		if (rtime) {
			this.hasRelativeTime = true;
			return setTimeout(this.renderTime.bind(this), 60000);
		}
	},

	renderBacklinks: function(model, links) {
		// No more backlinks, because posts deleted or something
		if (!links && this.$backlinks)
			return this.$backlinks.remove();
		if (!this.$backlinks) {
			this.$backlinks = $('<small/>')
				.insertAfter(this.$el.children('blockquote'))
		}
		var html = 'Replies:',
			// Link points to different thread
			diff;
		const num = this.model.get('num'),
			op = this.model.get('op') || num;
		for (var key in links) {
			if (!links.hasOwnProperty(key))
				continue;
			diff = links[key] != (op);
			html += common.parseHTML
				` <a class="history" href="${diff && links[key]}#${key}">
					&gt;&gt;${key}${diff && ' →'}
				</a>`;
		}
		this.$backlinks.html(html);
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
		main.openPostBox(num);
		main.postForm.addReference(num, sel);
	}
};

