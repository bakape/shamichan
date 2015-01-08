/* Thumbnail and image renderring */

var Hidamari = function(parent){
	this.model = parent.model;
	this.img = this.model.get('image');
	// Nothing to draw
	if (!this.img)
		return;
	this.$el = parent.$el;
	this.$fig = this.$el.children('figure');
	this.$a = this.$fig.children('a');
	this.$img = this.$a.children('img, video');
};


// Change thumbnail style
Hidamari.prototype.style = function(type){
	// Shitty hack
	// TODO: remove all, when options model is rewritten
	oneeSama.thumbStyle = type;
	this.render(type == 'hide');
};

// Rerenders the entire thumbnail, which can pretty expensive in batch,
// but good enough for now
Hidamari.prototype.render = function(hide, contract){
	if (hide){
		this.$fig.find('.imageSrc').text('[Show]');
		this.model.set({imageExpanded: false, thumbnailRevealed: false});
		return this.$img.remove();
	}
	// Don't replace expanded images, unless contracting
	if (this.model.get('imageExpanded') && !contract)
		return;
	this.$fig.find('.imageSrc').text(this.img.src);
	var $img = $(flatten(oneeSama.gazou_img(this.img, this.$el.is('section')).html).join(''));
	this.$a.remove();
	$img.appendTo(this.$fig);
	this.model.set({imageExpanded: false, thumbnailRevealed: true});
};

// Toggle spoiler
Hidamari.prototype.spoiler = function(toggle){
	if (options.get('thumbs') == 'hide')
		return;
	oneeSama.spoilToggle = toggle;
	this.render();
};

// Toggle animated GIF thumbnails
Hidamari.prototype.autogif = function(toggle){
	if (!/\.gif$/i.test(this.img.src) || options.get('thumbs') == 'hide')
		return;
	oneeSama.autoGif = toggle;
	this.render();
};

// Reveal hidden thumbnail by clicking [Show]
Hidamari.prototype.reveal = function (e){
	if (options.get('thumbs') != 'hide')
		return;
	e.preventDefault();
	var revealed = this.model.get('thumbnailRevealed');
	
	var self = this;
	with_dom(function(){
		self.render(revealed);
	});
	self.$fig.find('.imageSrc').text(revealed ? '[Show]' : '[Hide]');
	self.model.set('thumbnailRevealed', !revealed);
};

Hidamari.prototype.toggleExpansion = function(e){
	if (options.get('inlinefit') == 'none' || e.which != 1)
		return;
	e.preventDefault();
	
	var self = this;
	with_dom(function(){
		if (self.model.get('imageExpanded') != true)
			self.getDims();
		else
			self.render(options.get('thumbs') == 'hide', true);
	});
};

Hidamari.prototype.getDims =  function(){
	this.src = this.$a.attr('href');
	this.width = this.img.dims[0];
	this.height = this.img.dims[1];
	var fit = options.get('inlinefit');
	if (fit == 'full')
		return this.expand();
	var both = fit == 'both';
	this.fitToWindow(both || fit == 'width', both || fit == 'height');
};

Hidamari.prototype.fitToWindow = function(widthFlag, heightFlag){
	var newWidth = this.width;
	var newHeight = this.height;
	var aspect = this.width / this.height;
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
		this.width = newWidth;
		this.height = newHeight;
	}
	this.expand();
};

Hidamari.prototype.expand = function(){
	this.$img.replaceWith($('<'+ (this.img.length ? 'video' : 'img') +'/>', {
		src: this.src,
		width: this.width,
		height: this.height,
		autoplay: true,
		loop: true,
	}));
	this.model.set('imageExpanded', true);
};
