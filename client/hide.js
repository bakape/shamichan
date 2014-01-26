(function () {

// How many days before forgetting that we hid a given post
// (Otherwise the cookie will balloon in size)
var EXPIRY = 14;

var Hidden = new Kioku('hide', EXPIRY);

oneeSama.hook('menuOptions', function (info) {
	// should bail out if we're posting in here...
	if (!(info.model instanceof Thread))
		return;
	info.options.push('Hide');
});

menuHandlers.Hide = function (num, $thread) {
	Hidden.write(num, Hidden.now());
	$thread.next('hr').andSelf().hide();
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
