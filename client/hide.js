(function () {
	// How many days before forgetting that we hid a given post
	// (Otherwise the cookie will balloon in size)
	var EXPIRY = 7;

	var Hidden = new Kioku('hide', EXPIRY);

	oneeSama.hook('menuOptions', function (info) {
		if (!info.model)
			return; // can't hide drafts
		if (postForm && postForm.model.id == info.model.id)
			return; // can't hide own post
		info.options.push('Hide');
	});

	menuHandlers.Hide = function (model, $post) {
		Hidden.write(model.id, Hidden.now());
		model.set('hide', true);
		Backbone.trigger('hide', model); // bit of a hack...
	};

	/* Options menu clear control */

	var $clear = $('<input>', {
		type: 'button',
		val: 'Clear hidden',
		css: {display: 'block', 'margin-top': '1em'},
		click: function () {
			Hidden.purge_all();
			$clear.hide();
		},
	});

	oneeSama.hook('initOptions', function ($opts) {
		$opts.append($clear);
	});

	oneeSama.hook('renderOptions', function ($opts) {
		$clear.toggle(!_.isEmpty(Hidden.read_all()));
	});

	Hidden.purge_expired_soon();
})();
