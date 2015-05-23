/*
 * Background controller. Wallpapers, proper fitting and video backgrounds
 */

let Backbone = require('backbone'),
	main = require('../main'),
	common = main.common,
	options = main.options;

let BackgroundView = Backbone.View.extend({
	model: new Backbone.Model({
		id: 'background'
	}),

	initialize: function() {
		main.comply('background:store', this.store.bind(this));
		if (options.get('userBG'))
			this.render(null, true);
		this.listenTo(options, 'change:userBG', this.render);
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
			};
			img.src = event.target.result;
			if (options.get('userBG'))
				self.render(null, true);
		};
	},

	render: function(model, toggle) {
		this.$el.empty();
		if (!toggle)
			return;
		this.$el.html(`<img src="${localStorage.background}">`);
	}
});

main.defer(function() {
	module.exports = new BackgroundView({
		el: document.getElementById('user_bg')
	});
});
