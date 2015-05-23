/*
 * Background controller. Wallpapers, proper fitting and video backgrounds
 */

let Backbone = require('backbone'),
	main = require('../main'),
	common = main.common,
	options = main.options,
	state = main.state;

let BackgroundView = Backbone.View.extend({
	model: new Backbone.Model({
		id: 'background'
	}),

	initialize: function() {
		main.comply('background:store', this.store.bind(this));
		this.render();
		this.listenTo(options, {
			'change:userBG': this.render,
			'change:illyaBGToggle': this.render,
			'change:illyaMuteToggle': this.render
		});
	},

	// Store image as dataURL in localStorage
	store: function(target) {
		let reader = new FileReader(),
			self = this;
		reader.readAsDataURL(target.files[0]);
		reader.onload = function(event) {
			let img = new Image();
			img.onload = function() {
				// Convert to JPEG
				let canvas = document.createElement("canvas");
				canvas.width = img.width;
				canvas.height = img.height;
				canvas
					.getContext('2d')
					.drawImage(img, 0, 0, img.width, img.height);
				localStorage.background = canvas.toDataURL('image/jpeg', 0.95);
				// Apply new background
				if (options.get('userBG'))
					self.render();
			};
			img.src = event.target.result;
		};
	},

	render: function() {
		this.$el.empty().css('background', 'none');
		if (options.get('illyaBGToggle') && state.hotConfig.get('ILLYA_DANCE'))
			this.renderIllya();
		else if (options.get('userBG'))
			this.renderBackground();
	},

	renderBackground: function() {
		const bg = localStorage.background;
		if (!bg)
			return;
		this.$el
			// Need to set in separate call, because CSS
			.css('background', `url(${bg}) no-repeat fixed center`)
			.css('background-size', 'cover');
	},

	renderIllya: function() {
		const urlBase = main.config.MEDIA_URL + 'illya.';
		this.$el.html(common.parseHTML
			`<video autoplay loop ${options.get('illyaMuteToggle') && 'muted'}>
				<source src="${urlBase + 'webm'}" type="video/webm">
				<source src="${urlBase + 'mp4'}" type="video/mp4">
			</video>`
		)
	}
});

main.defer(function() {
	module.exports = new BackgroundView({
		el: document.getElementById('user_bg')
	});
});
