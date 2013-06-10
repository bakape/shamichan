var _ = require('../lib/underscore'),
    async = require('async'),
    caps = require('../server/caps'),
    fs = require('fs'),
    render = require('../server/render'),
    db = require('../db');

var DUMP_IDENT = {ip: '127.0.0.1', auth: 'dump'};

function Dumper(reader, out) {
	this.out = out;
	this.reader = reader;
	_.bindAll(this);
	reader.on('thread', this.on_thread);
	reader.on('post', this.on_post);
	reader.on('endthread', this.on_endthread);
}
var D = Dumper.prototype;

D.on_thread = function (op_post) {
	if (this.needComma) {
		this.out.write(',\n');
		this.needComma = false;
	}
	this.out.write('[\n\t' + JSON.stringify(op_post));
};

D.on_post = function (post) {
	this.out.write(',\n\t' + JSON.stringify(post));
};

D.on_endthread = function () {
	this.out.write('\n]');
	this.needComma = true;
};

D.destroy = function () {
	this.reader.removeListener('thread', this.on_thread);
	this.reader.removeListener('post', this.on_post);
	this.reader.removeListener('endthread', this.on_endthread);
	this.reader = null;
	this.out = null;
};

function dump_thread(op, board, ident, outputs, cb) {
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
		var dumper = new Dumper(reader, outputs.json);

		var out = outputs.html;
		render.write_thread_head(out, board, op, preThread.subject);

		var fakeReq = {ident: ident, headers: {}};
		var opts = {fullPosts: true, board: board};
		render.write_thread_html(reader, fakeReq, out, opts);

		reader.once('end', function () {
			outputs.json.write('\n');
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

function close_stream(stream, cb) {
	if (!stream.writable)
		return cb(null);
	if (stream.write(''))
		close();
	else
		stream.once('drain', close);

	function close() {
		try { stream.end(); } catch (e) {}
		cb(null);
	}
}

function load_state(cb) {
	require('../server/state').reload_hot_resources(cb);
}

if (require.main === module) (function () {
	var op = parseInt(process.argv[2], 10), board = process.argv[3];
	if (!op || !board) {
		console.error('Usage: node upkeep/dump.js <thread> <board>');
		process.exit(-1);
	}

	console.log('Loading state...');
	load_state(function (err) {
		if (err)
			throw err;
		console.log('Dumping thread...');

		var outputs = {
			json: process.stdout,
			html: fs.createWriteStream('dump.html'),
		};
		var streams = [outputs.json, outputs.html];

		dump_thread(op, board, DUMP_IDENT, outputs, function (err) {
			if (err)
				throw err;

			async.each(streams, close_stream, function (err) {
				// crappy flush for stdout (can't close it)
				if (process.stdout.write(''))
					process.exit(0);
				else
					process.stdout.on('drain', function () {
						process.exit(0);
					});
			});

		});
	});
})();
