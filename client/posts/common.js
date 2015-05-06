/*
 * Common methods to both OP and regular posts
 */
var $ = require('jquery'),
	common = require('../../common'),
	imager = require('./imager'),
	main = require('../main'),
	options = require('../options'),
	posting = require('./posting'),
	state = require('../state'),
	time = require('../time');

module.exports = {
	events: {
		'click >figure>figcaption>.imageSrc': 'revealThumbnail',
		'click >figure>a': 'imageClicked',
		'click >header>nav>a.quote': 'quotePost'
	},

	initCommon: function(){
		this.listenTo(this.model, {
			'change:hide': this.renderHide,
			'spoiler': this.renderSpoiler,
			'change:image': this.renderImage
		});
		this.listenTo(options, {
			'change:thumbs': this.changeThumbnailStyle,
			'change:spoilers': this.toggleSpoiler,
			'change:autogif': this.toggleAutogif,
			'change:anonymise': this.toggleAnonymisation,
			'change:rtime': this.renderTime
		});
		// Automatic image expansion
		this.listenTo(imager.massExpander, {
			'change:expand': this.toggleImageExpansion
		});
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
			html += common.html
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
		posting.openPostBox(num);
		main.postForm.addReference(num, sel);
	}
};
