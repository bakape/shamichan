/*
 R/a/dio API handler
 */

var common = require('../common'),
	hook =require('../util/hooks').hook,
	OK = require('./okyaku'),
	request = require('request');

var json,
	songMap = [
		[/Girls,? Be Ambitious/i, 'Joe'],
		[/Super Special/i, 'Super Special'],
		[/Turning Japanese/i, '\u540D\u7121\u3057'],
		[/Make a Man Out of You|Be a Man/i, 'Cute Little Girl']
	];

exports.name = null;

function parse(main) {
	let data = {
		np: main.np || '',
		listeners: main.listeners,
		dj: main.dj && main.dj.djname
	};
	// Stringify new object, so it can be compared to the old one
	let newJSON = JSON.stringify(data);
	if (newJSON != json) {
		json = newJSON;
		// Push new radio info to clients
		OK.push([0, common.RADIO, json]);
	}

	// Test song name against regex
	let name;
	for (let i = 0, l = songMap.length; i < l; i++) {
		let song = songMap[i];
		if (!song[0].test(data.np))
			continue;
		// Assign name replacement
		name = song[1];
		break;
	}

	// Build song queue
	var queue = '';
	for (let i = 0, l = main.queue.length; i < l; i++) {
		if (i > 0)
			queue += ' / ';
		queue +=  main.queue[i].meta;
	}

	exports.name = name;
	exports.queue = queue;
}

// Send r/a/dio banner on client sync
hook('clientSynced', function(info, cb) {
	info.client.send([0, common.RADIO, json]);
	cb(null);
});

// Send data to client on request
OK.dispatcher[common.RADIO] = function(msg, client) {
	client.send([0, common.RADIO, json]);
	return true;
};

function fetch() {
	// Query r/a/dio API
	request.get({
			url: 'https://r-a-d.io/api',
			json: true
		},
		function (err, resp, json) {
			if (err || resp.statusCode != 200 || !json || !json.main)
				return exports.name = exports.queue = null;
			parse(json.main);
		}
	);
}
fetch()
setInterval(fetch, 10000)
