/*
 * Thumbnail and image renderring
 */

import View from '../view'
import {$threads} from '../state'

/**
 * Thumbnail and image renderring logic
 */
class Imager extends View {
    /**
     * Construct a new post image handler
     * @param {Object} args
     */
    constructor(args) {
        super(args)
    }
}

exports.Hidamari = Backbone.View.extend({
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
		const reveal = arg === true,
			{model, el} = this;
		if (!image || !image.src)
			image = model.get('image');
		const figure = el.query('figure');
		if (figure)
			figure.remove();

		// Remove image on mod deletion
		if (!image)
			return;
		el.query('blockquote')
			.before(util.parseDOM(oneeSama.image(image, reveal)));

		// Scroll the post back into view, if contracting images taller than
		// the viewport
		if (manual && model.get('tallImage')) {
			window.scrollTop = el.getBoundingClientRect().top
				+ document.body.scrollTop
				- document.query('#banner').height;
		}

		model.set({
			// Only used in hidden thumbnail mode
			thumbnailRevealed: reveal,
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
			return this;
		this.toggleImageExpansion(true, img);
		return this;
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
			return window.open(oneeSama.imagePaths().src + img.src, '_blank');

		// Audio controls are always the same height and do not need to be
		// fitted
		if (img.ext === '.mp3')
			return this.renderAudio(img);

		let newWidth, newHeight,
			width = newWidth = img.dims[0],
			height = newHeight = img.dims[1];
		if (fit === 'full')
			return this.expandImage(img, {width, height});

		const both = fit === 'both',
			widthFlag = both || fit === 'width',
			heightFlag = both || fit === 'height',
			aspect = width / height;
		let fullWidth, fullHeight;
		if (widthFlag) {
			const maxWidth = this.imageMaxWidth();
			if (newWidth > maxWidth) {
				newWidth = maxWidth;
				newHeight = newWidth / aspect;
				fullWidth = true;
			}
		}
		if (heightFlag) {
			let maxHeight = window.innerHeight
				- document.query('#banner').offsetHeight;
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
		this.expandImage(img, width, height, fullHeight && !fullWidth);
	},
	// Calculate maximum horizontal dimension an image can be expanded to
	imageMaxWidth() {
		const {el, model} = this;
		return window.innerWidth
			- parseInt(el.closest('section').getBoundingClientRect().left) * 2
			- util.outerWidth(model.get('op') ? el : el.query('.background'));
	},
	expandImage(img, width, height, noMargin) {
		const isVideo = img.ext === '.webm';
		const attrs = {
			src: oneeSama.imagePaths().src + img.src,
			width,
			height
		};
		let cls = 'expanded';
		if (noMargin)
			cls += ' noMargin';
		attrs.class = cls;

		if (isVideo)
			attrs.autoplay = attrs.loop = attrs.controls = true

		this.el.query('figure').lastChild.innerHTML = common.parseHTML
			`<${isVideo ? 'video' : 'img'} ${attrs}>`;
		this.model.set({
			imageExpanded: true,
			tallImage: height > window.innerHeight
		});
	},
	renderAudio(img) {
		this.el.query('figure').append(util.parseDOM(common.parseHTML
			`<audio src="${oneeSama.imagePaths().src + img.src}"
				width="300"
				height="3em"
				autoplay loop controls
			>
			</audio>`));
		this.model.set('imageExpanded', true);
	}
});

// Expand all images
const ExpanderModel = Backbone.Model.extend({
	initialize() {
		$threads.on('click', '#expandImages', e => {
			e.preventDefault();
			this.toggle();
		});
	},
	toggle() {
		const expand = !this.get('expand');
		this.set('expand', expand).massToggle(expand);
		$threads
			.find('#expandImages')
			.text(main.lang.expander[+expand]);
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

const massExpander = exports.massExpander = new ExpanderModel();
main.reply('massExpander:unset', () => massExpander.unset());

// Proxy image clicks to views. More performant than dedicated listeners for
// each view.
$threads.on('click', 'img, video', function(e) {
	if (options.get('inlinefit') == 'none' || e.which !== 1)
		return;
	let model = util.getModel(e.target);
	if (!model)
		return;
	e.preventDefault();

	// Remove image hover preview, if any
	main.request('imager:clicked');
	model.dispatch('toggleImageExpansion', !model.get('imageExpanded'),
		model.get('image'), true);
});

// Reveal/hide thumbnail by clicking [Show]/[Hide] in hidden thumbnail mode
$threads.on('click', '.imageToggle', function(e) {
	e.preventDefault();
	let model = util.getModel(e.target);
	if (!model)
		return;
	main.follow(() =>
		model.dispatch('renderImage', !model.get('thumbnailRevealed')));
});
