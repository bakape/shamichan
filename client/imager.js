/* Thumbnail and image renderring */

var Hidamari = {
	events: {
		'click >figure>figcaption>.imageSrc': 'revealThumbnail',
		'click >figure>a': 'toggleImageExpansion',
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
		oneeSama.thumbStyle = type;
		this.renderThumbnail(type == 'hide');
	},
	// Rerenders the entire thumbnail, which can pretty expensive in batch,
	// but good enough for now
	renderThumbnail: function(hide, contract){
		var $fig = this.$el.children('figure');
		var $a = $fig.children('a');
		var $img = $a.children('img, video');
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
		$img = $(flatten(oneeSama.gazou_img(img, this.$el.is('section')).html).join(''));
		$a.remove();
		$img.appendTo($fig);
		this.model.set({imageExpanded: false, thumbnailRevealed: !hide});
	},

	toggleSpoiler: function(model, toggle){
		if (!this.model.has('image') || options.get('thumbs') == 'hide')
			return;
		oneeSama.spoilToggle = toggle;
		this.renderThumbnail();
	},
	// Toggle animated GIF thumbnails
	toggleAutogif: function(model, toggle){
		var img = this.model.get('image');
		if (!img || !/\.gif$/i.test(img.src) || options.get('thumbs') == 'hide')
			return;
		oneeSama.autoGif = toggle;
		this.renderThumbnail();
	},
	// Reveal hidden thumbnail by clicking [Show]
	revealThumbnail: function(e){
		if (options.get('thumbs') != 'hide')
			return;
		e.preventDefault();
		var revealed = this.model.get('thumbnailRevealed');
		this.renderThumbnail(revealed);
		this.$el.children('figure').find('.imageSrc').text(revealed ? '[Show]' : '[Hide]');
	},
	
	toggleImageExpansion: function(e){
		var img = this.model.get('image');
		var fit = options.get('inlinefit');
		if (!img || fit == 'none' || e.which != 1)
			return;
		e.preventDefault();
		if (this.model.get('imageExpanded') != true)
			this.fitImage(img, fit);
		else
			this.renderThumbnail(options.get('thumbs') == 'hide', true);
	},

	fitImage: function(img, fit){
		var width = newWidth = img.dims[0];
		var height = newHeight = img.dims[1];
		var video = !!img.length;
		if (fit == 'full')
			return this.expandImage(width, height, video);
		var both = fit == 'both';
		var widthFlag = both || fit == 'width';
		var heightFlag = both || fit == 'height';
		var aspect = width / height;
		if (widthFlag){
			var maxWidth = $(window).width() -
					this.$el.closest('section')[0].getBoundingClientRect().left*2;
			if (this.$el.is('article'))
				maxWidth -= this.$el.outerWidth() - this.$el.width();
			if (newWidth > maxWidth){
				newWidth = maxWidth;
				newHeight = newWidth / aspect;
			}
		}
		if (heightFlag){
			var maxHeight = $(window).height() - $('#banner').outerHeight();
			if (newHeight > maxHeight){
				newHeight = maxHeight;
				newWidth = newHeight * aspect;
			}
		}
		if (newWidth > 50 && newHeight > 50){
			width = newWidth;
			height = newHeight;
		}
		this.expandImage(width, height, video);
	},

	expandImage: function(width, height, video){
		var $fig = this.$el.children('figure');
		$fig.find('img, video').replaceWith($('<'+ (video ? 'video' : 'img') +'/>', {
			src: $fig.find('.imageSrc').attr('href'),
			width: width,
			height: height,
			autoplay: true,
			loop: true,
		}));
		this.model.set('imageExpanded', true);
	},
};
