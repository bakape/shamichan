/*
 * Thumbnail and image renderring
 */

let main = require('../main'),
	{$, $threads, Backbone, common, etc, oneeSama, options, state} = main;

let Hidamari = exports.Hidamari = {
	/*
	 Render entire <figure>. Rerenderring completely each time is considerable
	 overhed, but the alternative is very convoluted logic. I don't really want
	 to attach a FSM to each view, just for image renderring.
	 */
	renderImage(arg, image, manual) {
		/*
		 All kinds of listeners call this method, so we need to ensure we
		 always get the appropriate image object.
		 */
		const reveal = arg === true;
		let model = this.model,
			$el = this.$el;
		if (!image || !image.src)
			image = model.get('image');
		$el.children('figure').remove();
		// Remove image on mod deletion
		if (!image)
			return;
		$el
			.children('header')
			[model.get('op') ? 'after' : 'before'](
				oneeSama.image(image, reveal)
			);

		// Scroll the post back into view, if contracting images taller than
		// the viewport
		if (manual && model.get('tallImage'))
			$(window).scrollTop($el.offset().top - $('#banner').height());
		model.set({
			// Only used in hidden thumbnail mode
			thumbnailRevealed: reveal || options.get('thumbs') === 'hidden',
			imageExpanded: false,
			tallImage: false
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
	toggleImageExpansion(expand, img, manual) {
		const fit = options.get('inlinefit');
		if (!img || fit === 'none')
			return;
		if (expand)
			this.fitImage(img, fit);
		else
			this.renderImage(null, img, manual);
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
		if (widthFlag) {
			let maxWidth = $(window).width()
				// We need to go wider
				- this.$el
					.closest('section')[0]
					.getBoundingClientRect()
					.left * (isArticle ? 1 : 2);
			if (isArticle)
				maxWidth -= this.$el.outerWidth() - this.$el.width() + 5;
			if (newWidth > maxWidth) {
				newWidth = maxWidth;
				newHeight = newWidth / aspect;
				fullWidth = true;
			}
		}
		if (heightFlag) {
			let maxHeight = $(window).height() - $('#banner').outerHeight();
			if (newHeight > maxHeight) {
				newHeight = maxHeight;
				newWidth = newHeight * aspect;
				fullHeight = true;
			}
		}
		if (newWidth > 50 && newHeight > 50) {
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
		const isVideo = img.ext === '.webm';
		this.$el
			.children('figure')
			.children('a')
			.html(common.parseHTML
				`<${isVideo ? 'video' : 'img'}~
					src="${oneeSama.imagePaths().src + img.src}"
					width="${opts.width}"
					height="${opts.height}"
					${isVideo && 'autoplay loop '}
					${
						// Android Chrome disables autoplay because retarded
						// users. Show controls, so you can manually tap Play
						isVideo && main.isMobile && 'controls '
					}
					class="expanded${opts.fullWidth && ' fullWidth'}"
				>`
			);
		this.model.set({
			imageExpanded: true,
			tallImage: opts.height > window.innerHeight
		});
	},
	renderAudio(img) {
		this.$el
			.children('figure')
			.append(common.parseHTML
				`<audio src="${oneeSama.imagePaths().src + img.src}"
					width="300"
					height="3em"
					autoplay loop controls
				>
				</audio>`
			);
		this.model.set('imageExpanded', true);
	}
};

// Expand all images
let ExpanderModel = Backbone.Model.extend({
	id: 'massExpander',
	initialize() {
		$threads.on('click', '#expandImages', (e) => {
			e.preventDefault();
			this.toggle();
		});
	},
	toggle() {
		const expand = !this.get('expand');
		this.set('expand', expand).massToggle(expand);
		$threads
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

// Proxy image clicks to views. More performant than dedicated listeners for
// each view.
$threads.on('click', 'img, video', function(e) {
	if (options.get('inlinefit') == 'none' || e.which !== 1)
		return;
	let model = etc.getModel(e.target);
	if (!model)
		return;
	e.preventDefault();
	// Remove image hover preview, if any
	main.command('imager:clicked');
	main.follow(() =>
		model.dispatch(
			'toggleImageExpansion',
			!model.get('imageExpanded'),
			model.get('image'),
			true
		)
	);
});

// Reveal/hide thumbnail by clicking [Show]/[Hide] in hidden thumbnail mode
$threads.on('click', '.imageToggle', function(e) {
	e.preventDefault();
	let model = etc.getModel(e.target);
	if (!model)
		return;
	main.follow(() =>
		model.dispatch('renderImage', !model.get('thumbnailRevealed'))
	);
});
