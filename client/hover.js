/*
 * Hover previews
 */

var $ = require('jquery'),
	Backbone = require('backbone'),
	main = require('./main'),
	options = require('./options');

// Centralised mousemove target tracking
var mousemove = exports.mousemove = new Backbone.Model({
	id: 'mousemove',
	target: null
});

if (!main.isMobile) {
	main.$doc.on('mousemove', function(e) {
		mousemove.set('target', e.target);
	});
}
var ImageHoverView = Backbone.View.extend({
	initialize: function() {
		this.listenTo(this.model,'change:target', this.check);
		this.listenTo(options, 'imageClicked', function() {
			this.$el.empty();
		});
	},

	check: function(model, target) {
		// Disabled in options
		if (!options.get('imageHover'))
			return;
		var $target = $(target);
		if (!$target.is('img') || $target.hasClass('expanded'))
			return this.$el.empty();
		const src = $target.closest('a').attr('href'),
			isWebm = /\.webm$/.test(src);
		// Nothing to preview for PDF or MP3
		if (/\.pdf$/.test(src)
			|| /\.mp3$/.test(src)
			|| (isWebm && !options.get('webmHover'))
		)
			return this.$el.empty();
		$(isWebm ? '<video/>' : '<img/>', {
			src: src,
			autoplay: true,
			loop: true
		}).appendTo(this.$el.empty());
	}
});

if (!main.isMobile) {
	new ImageHoverView({
		model: mousemove,
		el: document.getElementById('hover_overlay')
	});
}
