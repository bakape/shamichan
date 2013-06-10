var caps = require('../server/caps'),
    render = require('../server/render'),
    db = require('../db');

var DUMP_IDENT = {ip: '127.0.0.1', auth: 'dump'};

function dump_thread(op, board, ident, out, cb) {
	if (!caps.can_access_board(ident, board))
		return cb(404);
	/*
	TODO: we don't have the thread DB loaded so this will fail
	if (!caps.can_access_thread(ident, op))
		return cb(404);
	*/

	var yaku = new db.Yakusoku(board, ident);
	var reader = new db.Reader(yaku);
	reader.get_thread(board, op, {});
	reader.once('nomatch', function () {
		cb(404);
		yaku.disconnect();
	});
	reader.once('redirect', function (op) {
		cb('redirect', op);
		yaku.disconnect();
	});
	reader.once('begin', function (preThread) {
		render.write_thread_head(out, board, op, preThread.subject);

		var fakeReq = {ident: ident, headers: {}};
		var opts = {fullPosts: true, board: board};
		render.write_thread_html(reader, fakeReq, out, opts);

		reader.once('end', function () {
			render.write_page_end(out, ident, true);
			yaku.disconnect();
			cb(null);
		});
	});

	function on_err(err) {
		yaku.disconnect();
		cb(err);
	}
	reader.once('error', on_err);
	yaku.once('error', on_err);
}

if (require.main === module) (function () {
	var op = parseInt(process.argv[2], 10), board = process.argv[3];
	if (!op || !board) {
		console.error('Usage: node upkeep/dump.js <thread> <board>');
		process.exit(-1);
	}
	console.log('Loading state...');
	require('../server/state').reload_hot_resources(function (err) {
		if (err)
			throw err;
		console.log('Dumping thread...');
		dump_thread(op, board, DUMP_IDENT, process.stdout,
				function (err) {
			if (err)
				throw err;

			// crappy flush()
			if (process.stdout.write(''))
				process.exit(0);
			else
				process.stdout.on('drain', function () {
					process.exit(0);
				});
		});
	});
})();
