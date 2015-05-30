/*
 * Background controller. Wallpapers, proper fitting and video backgrounds
 */

let $ = require('jquery'),
	Backbone = require('backbone'),
	// External, so relative to the root of the project
	blur = require('stack-blur'),
	main = require('../main'),
	common = main.common,
	options = main.options,
	state = main.state;

let BackgroundView = Backbone.View.extend({
	model: new Backbone.Model({
		id: 'background'
	}),

	initialize: function() {
		this.$css = $('#backgroundCSS');
		this.render();

		main.comply('background:store', this.store, this);
		this.listenTo(options, {
			'change:userBG': this.render,
			'change:illyaBGToggle': this.render,
			'change:illyaMuteToggle': this.render,
			'change:theme': this.render
		});
	},

	// Store image as dataURL in localStorage
	store: function(target) {
		// This could take a while, so loading animation
		main.command('loading:show');
		let reader = new FileReader();
		reader.readAsDataURL(target.files[0]);
		reader.onload = event => {
			let img = new Image();
			img.onload = () => {
				// Convert to JPEG
				let canvas = document.createElement("canvas");
				canvas.width = img.width;
				canvas.height = img.height;
				canvas
					.getContext('2d')
					.drawImage(img, 0, 0, img.width, img.height);
				localStorage.background = canvas.toDataURL('image/jpeg', 0.95);

				// Generate a blurred version of the background to use for
				// posts, modals, etc.
				blur.canvas(canvas, 0, 0, img.width, img.height, 10);
				localStorage.blurred = canvas.toDataURL('image/jpeg', 0.95);

				main.command('loading:hide');

				// Apply new background
				if (options.get('userBG'))
					this.render();
			};
			img.src = event.target.result;
		};
	},

	render: function() {
		this.$el.empty().css('background', 'none');
		this.$css.empty();
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
		// Add blurred background image to elements, if theme is glass or ocean
		const theme = options.get('theme');
		if (theme !== 'glass' && theme !== 'ocean')
			return;
		const blurred = localStorage.blurred;
		if (!blurred)
			return;
		this.$css.html(theme === 'glass' ? this.blurredGlass(blurred)
			: this.blurredOcean(blurred)
		);
	},

	blurredGlass: function(blurred) {
		const normal = 'rgba(40, 42, 46, 0.5)',
			editing = 'rgba(145, 145, 145, 0.5)';
		return common.parseHTML
			`article, aside, .pagination, .popup-menu, .modal, .bmodal,
				.preview, #banner
			{
				background:
					linear-gradient(${normal}, ${normal}),
					url(${blurred}) center fixed no-repeat;
				background-size: cover;
			}
			.editing {
				background:
					linear-gradient(${editing}, ${editing}),
					url(${blurred}) center fixed no-repeat;
				background-size: cover;
			}`
	},

	blurredOcean: function(blurred) {
        	const normal = 'rgba(28, 29, 34, 0.781)',
                      editing = 'rgba(44, 57, 71, 0.88)';

		return common.parseHTML
			`article, aside, .pagination, .popup-menu, .modal, .bmodal,
				.preview, #banner
			{
				background:
					linear-gradient(${normal}, ${normal}),
					url(${blurred}) center fixed no-repeat;
				background-size: cover;
			}
			.editing {
				background:
					linear-gradient(${editing}, ${editing}),
					url(${blurred}) center fixed no-repeat;
				background-size: cover;
			}`;
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
