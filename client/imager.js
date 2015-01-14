/* Thumbnail and image renderring */

var Hidamari = {
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

	imageClicked: function(e){
		if (options.get('inlinefit') == 'none' || e.which != 1)
			return;
		e.preventDefault();
		this.toggleImageExpansion();
	},

	autoExpandImage: function(){
		var img = this.model.get('image');
		// Don't autoexpand WebM
		if (img && massExpander.get('expand') && !/\.webm$/i.test(img.src))
			this.toggleImageExpansion();
	},

	toggleImageExpansion: function(){
		var img = this.model.get('image');
		var fit = options.get('inlinefit');
		if (!img || fit == 'none')
			return;
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
		var isArticle = this.$el.is('article');
		var fullWidth, fullHeight;
		if (widthFlag){
			var maxWidth = $(window).width() -
					// We have to go wider
					this.$el.closest('section')[0].getBoundingClientRect().left* (isArticle ? 1 : 2);
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
		this.expandImage(width, height, video, fullWidth && !fullHeight);
	},

	expandImage: function(width, height, video, fullWidth){
		var $fig = this.$el.children('figure');
		$fig.find('img, video').replaceWith($('<'+ (video ? 'video' : 'img') +'/>', {
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
};

var Shell = Backbone.Model.extend();

var massExpander = new Shell({
	expand: false
});

$('#expandImages').click(function(e){
	e.preventDefault();
	var expand = massExpander.get('expand');
	$(e.target).text((expand ? 'Expand' : 'Contract')+' Images');
	massExpander.set('expand', !expand);
});
