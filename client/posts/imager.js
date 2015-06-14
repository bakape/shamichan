/*
 * Thumbnail and image renderring
 */

let main = require('../main'),
	{$, Backbone, common, etc, oneeSama, options, state} = main;

let Hidamari = exports.Hidamari = {
	/*
	 Render entire <figure>. Rerenderring completely each time is considerable
	 overhed, but the alternative is very convoluted logic. I don't really want
	 to attach a FSM to each view, just for image renderring.
	 */
	renderImage(arg, image) {
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
		this.$el
			.children('header')
			[this.model.get('op') ? 'after' : 'before'](
				oneeSama.image(image, reveal)
			);
		this.model.set({
			// Only used in hidden thumbnail mode
			thumbnailRevealed: reveal || options.get('thumbs') === 'hidden',
			imageExpanded: false
		});
	},
	autoExpandImage() {
		const img = this.model.get('image');
		if (!img
			|| !massExpander.get('expand')
			// Don't auto expand webm/PDF/MP3
			|| ['.webm', '.pdf', '.mp3'].indexOf(img.ext) > -1
		)
			return;
		this.toggleImageExpansion(true, img);
	},
	// Reveal/hide thumbnail by clicking [Show]/[Hide] in hidden thumbnail mode
	toggleThumbnailVisibility(e) {
		e.preventDefault();
		main.follow(() =>
			this.renderImage(!this.model.get('thumbnailRevealed'))
		);
	},
	imageClicked(e){
		if (options.get('inlinefit') == 'none' || e.which !== 1)
			return;
		// Remove image hover preview, if any
		options.trigger('imageClicked');
		e.preventDefault();
		main.follow(() =>
			this.toggleImageExpansion(!this.model.get('imageExpanded'))
		);
	},
	toggleImageExpansion(expand, img = this.model.get('image')) {
		const fit = options.get('inlinefit');
		if (!img || fit === 'none')
			return;
		if (expand)
			this.fitImage(img, fit);
		else
			this.renderImage(null, img);
	},
	fitImage(img, fit){
		// Open PDF in a new tab on click
		if (img.ext === '.pdf')
			return window.open(oneeSama.imagePaths().src + img.src,
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
	expandImage(img, opts) {
		const tag = (img.ext === '.webm') ? 'video' : 'img';
		this.$el
			.children('figure')
			.children('a')
			.html(common.parseHTML
				`<${tag}~
					src="${oneeSama.imagePaths().src + img.src}"
					width="${opts.width}"
					height="${opts.height}"
					autoplay="true"
					loop="true"
					class="expanded${opts.fullWidth && ' fullWidth'}"
				>`
			);
		this.model.set('imageExpanded', true);
	},
	renderAudio(img) {
		this.$el
			.children('figure')
			.children('a')
			.append(common.parseHTML
				`<audio src="${oneeSama.imagePaths().src + img.src}"
					width="300"
					height="3em"
					autoplay="true"
					loop="true"
					controls="true"
				>
				</audio>`
			);
		this.model.set('imageExpanded', true);
	},
	// Minimal image thumbnail swap for lazy loading
	loadImage(image) {
		let el = this.el
			.getElementsByTagName('figure')[0]
			.getElementsByTagName('img')[0];
		el.outerHTML = oneeSama.thumbnail(image);
	}
};

// Expand all images
let ExpanderModel = Backbone.Model.extend({
	id: 'massExpander',
	initialize() {
		main.$threads.on('click', '#expandImages', (e) => {
			e.preventDefault();
			this.toggle();
		});
	},
	toggle() {
		const expand = !this.get('expand');
		this.set('expand', expand).massToggle(expand);
		main.$threads
			.find('#expandImages')
			.text(`${expand ? 'Contract' : 'Expand'} Images`);
	},
	// More efficent than individual listeners
	massToggle(expand) {
		const fit = options.get('inlinefit');
		if (fit === 'none')
			return;
		let models = state.posts.models;
		for (let i = 0, l = models.length; i < l; i++) {
			let model = models[i],
				img = model.get('image');
			if (!img)
				continue;
			if (expand)
				model.dispatch('fitImage', img, fit);
			else
				model.dispatch('renderImage', null, img);
		}
	}
});

let massExpander = exports.massExpander = new ExpanderModel();
main.comply('massExpander:unset', () => massExpander.unset());

// Lazy load images with less UI locking
function loadImages() {
	if (options.get('thumbs') === 'hide')
		return;
	etc.defferLoop(state.posts.models, function(model) {
		const image = model.get('image');
		if (!image)
			return;
		model.dispatch('loadImage', image);
	})
}
main.comply('imager:lazyLoad', loadImages);
