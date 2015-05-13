/*
 * Thumbnail and image renderring
 */

'use strict';

let $ = require('jquery'),
	Backbone = require('backbone'),
	common = require('../../common'),
	main = require('../main'),
	options = require('../options');

let Hidamari = exports.Hidamari = {
	/*
	 Render entire <figure>. Rerenderring completely each time is considerable
	 overhed, but the alternative is very convoluted logic. I don't really want
	 to attach a FSM to each view, just for image renderring.
	 */
	renderImage: function(arg, image) {
		/*
		 All kinds of listeners call this method, so we need to ensure we
		 always get the appropriate image object.
		 */
		const reveal = arg === true;
		if (!image || !image.src)
			image = this.model.get('image');
		this.$el.children('figure').remove();
		// Remove image on mod deletion
		if (!image)
			return;
		const html = common.flatten(main.oneeSama.gazou(image, reveal))
			.join('');
		let $header = this.$el.children('header');
		if (this.model.get('op'))
			// A post
			$header.after(html);
		else
			// A thread
			$header.before(html);

		this.model.set({
			// Only used in hidden thumbnail mode
			thumbnailRevealed: reveal || options.get('thumbs') === 'hidden',
			imageExpanded: false
		});
	},

	autoExpandImage: function() {
		const img = this.model.get('image');
		if (!img
			|| !massExpander.get('expand')
			// Don't auto expand webm/PDF/MP3
			|| ['.webm', '.pdf', '.mp3'].indexOf(img.ext) > -1
		)
			return;
		this.toggleImageExpansion(true, img);
	},

	renderSpoiler: function(spoiler) {
		let img = this.model.get('image');
		img.spoiler = spoiler;
		this.renderImage(img);
	},

	toggleSpoiler: function() {
		const img = this.model.get('image');
		if (!img || !img.spoiler)
			return;
		this.renderImage(img);
	},

	// Toggle animated GIF thumbnails
	toggleAutogif: function() {
		const img = this.model.get('image');
		if (!img || img.ext !== '.gif')
			return;
		this.renderImage(img);
	},

	// Reveal/hide thumbnail by clicking [Show]/[Hide] in hidden thumbnail mode
	toggleThumbnailVisibility: function(e) {
		e.preventDefault();
		this.renderImage(!this.model.get('thumbnailRevealed'));
	},

	imageClicked: function(e){
		if (options.get('inlinefit') == 'none' || e.which !== 1)
			return;
		// Remove image hover preview, if any
		options.trigger('imageClicked');
		e.preventDefault();
		this.toggleImageExpansion(!this.model.get('imageExpanded'));
	},

	toggleImageExpansion: function(expand, img = this.model.get('image')) {
		const fit = options.get('inlinefit');
		if (!img || fit === 'none')
			return;
		if (expand)
			this.fitImage(img, fit);
		else
			this.renderImage(null, img);
	},

	fitImage: function(img, fit){
		// Open PDF in a new tab on click
		if (img.ext === '.pdf')
			return window.open(main.oneeSama.imagePaths().src + img.src,
				'_blank'
			);
		// Audio controls are always the same height and do not need to be
		// fitted
		if (img.ext === '.mp3')
			return this.renderAudio(img);
		let newWidth, newHeight,
			width = newWidth = img.dims[0],
			height = newHeight = img.dims[1];
		if (fit === 'full') {
			return this.expandImage(img, {
				width,
				height
			});
		}
		const both = fit === 'both',
			widthFlag = both || fit === 'width',
			heightFlag = both || fit === 'height',
			aspect = width / height,
			isArticle = !!this.model.get('op');
		let fullWidth, fullHeight;
		if (widthFlag){

			let maxWidth = $(window).width()
				// We need to go wider
				- this.$el
					.closest('section')[0]
					.getBoundingClientRect()
					.left * (isArticle ? 1 : 2);
			if (isArticle)
				maxWidth -= this.$el.outerWidth() - this.$el.width() + 5;
			if (newWidth > maxWidth){
				newWidth = maxWidth;
				newHeight = newWidth / aspect;
				fullWidth = true;
			}
		}
		if (heightFlag){
			let maxHeight = $(window).height() - $('#banner').outerHeight();
			if (newHeight > maxHeight){
				newHeight = maxHeight;
				newWidth = newHeight * aspect;
				fullHeight = true;
			}
		}
		if (newWidth > 50 && newHeight > 50){
			width = newWidth;
			height = newHeight;
		}
		this.expandImage(img, {
			width,
			height,
			fullWidth: fullWidth && !fullHeight
		});
	},

	expandImage: function(img, opts) {
		const tag = (img.ext === '.webm') ? 'video' : 'img';
		this.$el
			.children('figure')
			.children('a')
			.html(common.parseHTML
				`<${tag}~
					src="${main.oneeSama.imagePaths().src + img.src}"
					width="${opts.width}"
					height="${opts.height}"
					autoplay="true"
					loop="true"
					class="expanded${opts.fullWidth && ' fullWidth'}"
				>`
			);
		this.model.set('imageExpanded', true);
	},

	renderAudio: function(img) {
		this.$el
			.children('figure')
			.children('a')
			.append(common.parseHTML
				`<audio
					src="${main.oneeSama.imagePaths().src + img.src}"
					width="300"
					height="3em"
					autoplay="true"
					loop="true"
					controls="true"
				>
				</audio>`
			);
		this.model.set('imageExpanded', true);
	}
};

// Expand all images
let ExpanderModel = Backbone.Model.extend({
	id: 'massExpander',

	initialize: function() {
		main.$threads.on('click', '#expandImages', function(e) {
			e.preventDefault();
			this.toggle();
		}.bind(this));
	},

	toggle: function() {
		const expand = !this.get('expand');
		this.set('expand', expand);
		main.$threads
			.find('#expandImages')
			.text(`${expand ? 'Contract' : 'Expand'} Images`);
	}
});

let massExpander = exports.massExpander = new ExpanderModel();
