(function () {

// How many days before forgetting that we hid a given post
// (Otherwise the cookie will balloon in size)
var EXPIRY = 14;

oneeSama.hook('menuOptions', function (info) {
	// TODO: use a model lookup for this check
	// should bail out if we're posting in here...
	var $thread = $('#' + info.num);
	if (!$thread.length || !$thread.is('section'))
		return;
	info.options.push('Hide');
});

menuHandlers.Hide = function (num, $thread) {
	var hidden = read_hidden();
	hidden[num] = Math.floor(new Date().getTime() / 1000);
	write_hidden(hidden);
	$thread.next('hr').andSelf().hide();
};

function read_hidden() {
	var hidden;
	try {
		hidden = JSON.parse(localStorage.getItem('hide'));
	}
	catch (e) {}
	return _.isObject(hidden) ? hidden : {};
}

function write_hidden(hidden) {
	if (_.isEmpty(hidden)) {
		localStorage.removeItem('hide');
		$.cookie('hide', null);
	}
	else {
		localStorage.setItem('hide', JSON.stringify(hidden));
		var nums = _.keys(hidden);
		nums.sort(function (a, b) {
			return parseInt(a, 10) - parseInt(b, 10);
		});
		$.cookie('hide', nums.join(','), {expires: EXPIRY});
	}
}

function expire_hidden() {
	var hidden = read_hidden();
	var now = new Date().getTime()/1000, expired = [];
	for (var num in hidden) {
		var time = hidden[num];
		if (now > time + 60*60*24*EXPIRY) {
			expired.push(num);
		}
	}
	if (expired.length) {
		_.forEach(expired, function (num) {
			delete hidden[num];
		});
		write_hidden(hidden);
	}
}

/* Options menu clear control */

var $clear = $('<input>', {
	type: 'button',
	val: 'Clear hidden',
	css: {display: 'block', 'margin-top': '1em'},
	click: function () {
		write_hidden();
		$clear.hide();
	},
});

oneeSama.hook('initOptions', function ($opts) {
	$opts.append($clear);
});

oneeSama.hook('renderOptions', function ($opts) {
	$clear.toggle(!_.isEmpty(read_hidden()));
});


setTimeout(expire_hidden, 9000);

})();
