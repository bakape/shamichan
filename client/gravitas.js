(function () {
	Backbone.on('afterInsert', function (model) {
		if (model.id != MILLION)
			return;

		if (!model.get('op'))
			gravitas_body();
	});

	ComposerView.prototype.add_own_gravitas = function (msg) {
		var $el = this.$el.addClass('gravitas');
		if (msg.image) {
			$el.css('background-image', oneeSama.gravitas_style(msg.image));
			var bg = $el.css('background-color');
			$el.css('background-color', 'black');
			setTimeout(function () { $el.css('background-color', bg); }, 500);
		}
		if (!this.model.get('op'))
			gravitas_body();
		this.blockquote.css({'margin-left': '', 'padding-left': ''});
	};

	if (window.gravitas)
		$(gravitas_body);
})();
