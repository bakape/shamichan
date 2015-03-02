var common = require('../common'),
	hook =require('../hooks').hook,
	push = require('../server/okyaku').push,
	request = require('request');

var	json,
	songMap = [
		[/Girls,? Be Ambitious/i, 'Joe'],
		[/Super Special/i, 'Super Special'],
		[/Turning Japanese/i, '\u540D\u7121\u3057'],
		[/Make a Man Out of You|Be a Man/i, 'Cute Little Girl'],
	];

exports.name = null;

function parse(main) {
	var data = {
		np: main.np || '',
		listeners: main.listeners,
		dj: main.dj && main.dj.djname
	};
	// Stringify new object, so it can be compared to the old one
	var newJSON = JSON.stringify(data);
	if (newJSON != json) {
		json = newJSON;
		// Push new radio info to clients
		push([0, common.RADIO, json]);
	}

	// Test song name against regex
	var name = null;
	for (var i of songMap) {
		if (!i[0].test(data.np))
			continue;
		// Assign name replacement
		name = i[1];
		break;
	}

	// Build song queue
	var queue = '';
	for (var i = 0; i < main.queue.length; i++) {
		if (i > 0)
			queue += ' | ';
		queue += main.queue[i].meta;
	}

	exports.name = name;
	exports.queue = queue;
}

// Send r/a/dio banner on client sync
hook('clientSynced', function(info, cb) {
	info.client.send([0, common.RADIO, json]);
	cb(null);
});

function fetch() {
	// Query r/a/dio API
	request.get({url: 'https://r-a-d.io/api', json: true,},
		function (err, resp, json){
			if (err || resp.statusCode != 200 || !json || !json.main) {
				exports.name = exports.queue = null;
				return again();
			}
			parse(json.main);
			again();
		}
	);
};

function again() {
	setTimeout(fetch, 10000);
}

fetch();