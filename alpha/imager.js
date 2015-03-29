/*
 * Thumbnail and image renderring
 */

var $ = require('jquery'),
	main = require('./main'),
	options = require('./options');

var Hidamari = exports.Hidamari = {
	events: {
		'click >figure>figcaption>.imageSrc': 'revealThumbnail',
		'click >figure>a': 'imageClicked',
	},

	renderSpoiler: function(spoiler){
		this.model.get('image').spoiler = spoiler;
		this.renderThumbnail();
	},

	changeThumbnailStyle: function(model, type){
		if (!this.model.has('image'))
			return;
		// Shitty hack
		// TODO: remove all, when options model is rewritten
		main.oneeSama.thumbStyle = type;
		this.renderThumbnail(type == 'hide');
	},

	// Rerenders the entire thumbnail, which can pretty expensive in batch,
	// but good enough for now
	renderThumbnail: function(hide, contract){
		var $fig = this.$el.children('figure'),
			$a = $fig.children('a'),
			$img = $a.children('img, video');
		if (hide === undefined)
			hide = options.get('thumbs') == 'hide';
		if (hide){
			$fig.find('.imageSrc').text('[Show]');
			this.model.set({imageExpanded: false, thumbnailRevealed: false});
			return $img.remove();
		}
		// Don't replace expanded images, unless contracting
		if (this.model.get('imageExpanded') && !contract)
			return;
		var img = this.model.get('image');
		$fig.find('.imageSrc').text(img.src);
		$img = $(
			common.flatten(
				main.oneeSama.gazou_img(img, this.$el.is('section')
			).html).join('')
		);
		$a.remove();
		$img.appendTo($fig);
		this.model.set({imageExpanded: false, thumbnailRevealed: !hide});
	},

	toggleSpoiler: function(model, toggle){
		if (!this.model.has('image') || options.get('thumbs') == 'hide')
			return;
		main.oneeSama.spoilToggle = toggle;
		this.renderThumbnail();
	},

	// Toggle animated GIF thumbnails
	toggleAutogif: function(model, toggle){
		const img = this.model.get('image');
		if (!img || img.ext !== '.gif' || options.get('thumbs') == 'hide')
			return;
		main.oneeSama.autoGif = toggle;
		this.renderThumbnail();
	},

	// Reveal hidden thumbnail by clicking [Show]
	revealThumbnail: function(e){
		if (options.get('thumbs') != 'hide')
			return;
		e.preventDefault();
		var revealed = this.model.get('thumbnailRevealed');
		this.renderThumbnail(revealed);
		this.$el.children('figure').find('.imageSrc')
			.text(revealed ? '[Show]' : '[Hide]');
	},

	imageClicked: function(e){
		if (options.get('inlinefit') == 'none' || e.which != 1)
			return;
		e.preventDefault();
		this.toggleImageExpansion();
	},

	autoExpandImage: function(){
		var expand = massExpander.get('expand');
		if (expand)
			this.toggleImageExpansion(null, expand);
	},

	toggleImageExpansion: function(model, expand){
		var img = this.model.get('image');
		const fit = options.get('inlinefit');
		if (!img || fit == 'none')
			return;
		// Don't autoexpand webm/PDF/MP3 with Expand All enabled
		if (expand !== undefined
			&& (img.ext == '.webm'
			|| img.ext == '.pdf'
			|| img.ext == '.mp3'))
				return;
		if  (expand != false)
			expand = expand || this.model.get('imageExpanded') != true;
		if (expand)
			this.fitImage(img, fit);
		else
			this.renderThumbnail(options.get('thumbs') == 'hide', true);
	},

	fitImage: function(img, fit){
		// Open PDF in a new tab on click
		if (img.ext == '.pdf')
			return window.open(mediaURL + 'src/' + img.src, '_blank');
		// Audio controls are always the same height and do not need to be fitted
		if (img.ext == '.mp3')
			return this.renderAudio();
		const width = newWidth = img.dims[0],
			height = newHeight = img.dims[1];
		if (fit == 'full')
			return this.expandImage(width, height, img.ext);
		const both = fit == 'both',
			widthFlag = both || fit == 'width',
			heightFlag = both || fit == 'height',
			aspect = width / height,
			isArticle = this.$el.is('article');
		var fullWidth, fullHeight;
		if (widthFlag){
			var maxWidth = $(window).width() -
				// We have to go wider
				this.$el.closest('section')[0].getBoundingClientRect().left
					* (isArticle ? 1 : 2);
			if (isArticle)
				maxWidth -= this.$el.outerWidth() - this.$el.width() + 5;
			if (newWidth > maxWidth){
				newWidth = maxWidth;
				newHeight = newWidth / aspect;
				fullWidth = true;
			}
		}
		if (heightFlag){
			var maxHeight = $(window).height() - $('#banner').outerHeight();
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
		this.expandImage(width, height, img.ext, fullWidth && !fullHeight);
	},

	expandImage: function(width, height, ext, fullWidth){
		var $fig = this.$el.children('figure'),
			tag;
		if (ext == '.webm')
			tag = 'video';
		else
			tag = 'img';
		$fig.find('img, video').replaceWith($('<'+ tag +'/>', {
			src: $fig.find('.imageSrc').attr('href'),
			width: width,
			height: height,
			autoplay: true,
			loop: true,
			// Even wider
			'class': 'expanded'+ (fullWidth ? ' fullWidth' : ''),
		}));
		this.model.set('imageExpanded', true);
	},

	renderAudio: function() {
		$a = this.$el.children('figure').children('a');
		$('<audio/>', {
			src: $a.attr('href'),
			width: 300,
			height: '3em',
			autoplay: true,
			loop: true,
			controls: true
		})
			.appendTo($a);
		this.model.set('imageExpanded', true);
	}
};

var massExpander = exports.massExpander = new Backbone.Model({
	expand: false
});

$('#expandImages').click(function(e){
	e.preventDefault();
	const expand = massExpander.get('expand');
	$(e.target).text((expand ? 'Expand' : 'Contract')+' Images');
	massExpander.set('expand', !expand);
});
