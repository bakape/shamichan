var _ = require('../lib/underscore'),
    db = require('../db'),
    caps = require('../server/caps'),
    winston = require('winston');

var CURFEW_IDENT = {auth: 'Curfew'};

function shutdown(board, cb) {
	var yaku = new db.Yakusoku(board, CURFEW_IDENT);
	yaku.teardown(board, function (err) {
		yaku.disconnect();
		cb(err);
	});
}

function at_next_curfew_start(board, func) {
	var when = caps.curfew_starting_time(board);
	winston.info('Next curfew for ' + board + ' at ' + when);
	setTimeout(func, when.getTime() - new Date().getTime());
}

function enforce(board) {
	at_next_curfew_start(board, function () {
		winston.info('Curfew ' + board + ' at ' + new Date());
		shutdown(board, function (err) {
			if (err)
				winston.error(err);
		});
		setTimeout(enforce.bind(null, board), 30 * 1000);
	});
}

if (require.main === module) {
	winston.info('Started at ' + new Date());
	require('../config').CURFEW_BOARDS.forEach(enforce);
}
