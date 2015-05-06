/*
 * Common methods to both OP and regular posts
 */
var $ = require('jquery'),
	imager = require('./imager'),
	main = require('../main'),
	options = require('../options'),
	posting = require('./posting'),
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
		// Models get added to multiple collections. Prevent duplication by
		// calling only once
		this.listenToOnce(this.model, {
			add: function() {
				this.renderRelativeTime();
				this.fun();
				// Anonymise on post insertion
				if (options.get('anonymise'))
					this.toggleAnonymisation(null, true);
			}
		});
		this.listenTo(options, {
			'change:thumbs': this.changeThumbnailStyle,
			'change:spoilers': this.toggleSpoiler,
			'change:autogif': this.toggleAutogif,
			'change:anonymise': this.toggleAnonymisation
		});
		this.listenTo(imager.massExpander, {
			'change:expand': this.toggleImageExpansion
		});
	},

	renderRelativeTime: function(){
		if (main.oneeSama.rTime){
			var $time = this.$el.find('time').first();
			const t = time.date_from_time_el($time[0]).getTime();
			var timer = setInterval(function(){
				$time.html(main.oneeSama.relative_time(t, new Date().getTime()));
			}, 60000);
			this.listenToOnce(this.model, 'removeSelf', function(){
				clearInterval(timer);
			});
		}
	},

	fun: function(){
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
