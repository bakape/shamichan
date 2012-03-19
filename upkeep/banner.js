var db = require('../db');

// The state management is completely busted right now since this is a
// one-process operation and the banner state is also in-process

var RADIO_IDENT = {auth: 'Radio'};

function update_banner(board, op, message, cb) {
	var yaku = new db.Yakusoku(board, RADIO_IDENT);
	yaku.set_banner(op, message, cb);
}

if (require.main === module) {
	var args = process.argv;
	if (args.length != 5)
		throw "Arguments board, op, message required."
	var op = parseInt(args[3], 10);
	update_banner(args[2], op, args[4], function (err) {
		if (err)
			console.error(err);
		process.exit(err ? -1 : 0);
	});
}
