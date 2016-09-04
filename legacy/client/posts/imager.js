exports.Hidamari = Backbone.View.extend({
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
