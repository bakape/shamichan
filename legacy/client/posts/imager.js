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

		model.set({
			// Only used in hidden thumbnail mode
			thumbnailRevealed: reveal,
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
		const fit = options.get('inlinefit')
		if (fit === 'none')
			return

		for (let model of state.posts.models) {
			const img = model.get("image")

			if (!img || img.audio) {
				continue
			}
			switch (img.ext) {
			case ".pdf":
			case ".mp3":
				continue
			}

			if (expand) {
				model.dispatch('fitImage', img, fit)
			} else {
				model.dispatch('renderImage', null, img)
			}
		}
	}
});

const massExpander = exports.massExpander = new ExpanderModel();
main.reply('massExpander:unset', () => massExpander.unset());

// Reveal/hide thumbnail by clicking [Show]/[Hide] in hidden thumbnail mode
$threads.on('click', '.imageToggle', function(e) {
	e.preventDefault();
	let model = util.getModel(e.target);
	if (!model)
		return;
	main.follow(() =>
		model.dispatch('renderImage', !model.get('thumbnailRevealed')));
});
