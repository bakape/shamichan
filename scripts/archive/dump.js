/* Dumps the given thread to JSON.
 * Not production-ready.
 */

var _ = require('underscore'),
    async = require('async'),
    caps = require('../../server/caps'),
    db = require('../../db'),
    etc = require('../../util/etc'),
    fs = require('fs'),
    joinPath = require('path').join,
    render = require('../../server/render'),
    util = require('util');

var DUMP_DIR = 'www/archive';
var AUTH_DUMP_DIR = 'authdump';

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
	this.op = op_post.num;
	this.out.write('[\n' + JSON.stringify(tweak_post(op_post)));
};

D.on_post = function (post) {
	this.out.write(',\n' + JSON.stringify(tweak_post(post, this.op)));
};

D.on_endthread = function () {
	this.out.write('\n]');
	this.needComma = true;
	this.op = null;
};

D.destroy = function () {
	this.reader.removeListener('thread', this.on_thread);
	this.reader.removeListener('post', this.on_post);
	this.reader.removeListener('endthread', this.on_endthread);
	this.reader = null;
	this.out = null;
};


function AuthDumper(reader, out) {
	Dumper.call(this, reader, out);
}
util.inherits(AuthDumper, Dumper);
var AD = AuthDumper.prototype;

AD.on_thread = function (post) {
	this.out.write('{"ips":{');

	if (post.num && post.ip) {
		this.out.write('"'+post.num+'":'+JSON.stringify(post.ip));
		this.needComma = true;
	}
};

AD.on_post = function (post) {
	if (post.num && post.ip) {
		if (this.needComma)
			this.out.write(',');
		else
			this.needComma = true;
		this.out.write('"'+post.num+'":'+JSON.stringify(post.ip));
	}
};

AD.on_endthread = function () {
	this.out.write('}}');
	this.needComma = false;
};

function tweak_post(post, known_op) {
	post = _.clone(post);

	/* thread-only */
	if (typeof post.tags == 'string')
		post.tags = db.parse_tags(post.tags);
	if (typeof post.origTags == 'string')
		post.origTags = db.parse_tags(post.origTags);
	if (typeof post.hctr == 'string')
		post.hctr = parseInt(post.hctr, 10);
	if (typeof post.imgctr == 'string')
		post.imgctr = parseInt(post.imgctr, 10);

	/* post-only */
	if (known_op == post.op)
		delete post.op;

	if (post.hideimg) {
		delete post.image;
		delete post.hideimg;
	}
	if (post.body == '')
		delete post.body;

	/* blacklisting is bad... */
	delete post.ip;

	return post;
}

function dump_thread(op, board, ident, outputs, cb) {
	if (!caps.can_access_board(ident, board))
		return cb(404);
	if (!caps.can_access_thread(ident, op))
		return cb(404);

	var yaku = new db.Yakusoku(board, ident);
	var reader = new db.Reader(ident);
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
		var authDumper = new AuthDumper(reader, outputs.auth);

		var out = outputs.html;
		render.write_thread_head(out, board, op, {
			subject: preThread.subject,
		});

		var fakeReq = {ident: ident, headers: {}};
		var opts = {fullPosts: true, board: board};
		render.write_thread_html(reader, fakeReq, out, opts);

		reader.once('end', function () {
			outputs.json.write('\n');
			outputs.auth.write('\n');
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
		// deal with process.stdout not being closable
		try {
			stream.destroySoon(function (err) {
				if (cb)
					cb(err);
				cb = null;
			});
		}
		catch (e) {
			if (cb)
				cb(null);
			cb = null;
		}
	}
}

function load_state(cb) {
	async.series([
		etc.checked_mkdir.bind(null, DUMP_DIR),
		etc.checked_mkdir.bind(null, AUTH_DUMP_DIR),
		require('../server/state').reload_hot_resources,
		db.track_OPs,
	], cb);
}

if (require.main === module) (function () {
	var op = parseInt(process.argv[2], 10), board = process.argv[3];
	if (!op) {
		console.error('Usage: node scripts/dump.js <thread>');
		process.exit(-1);
	}

	console.log('Loading state...');
	load_state(function (err) {
		if (err)
			throw err;

		if (!board)
			board = db.first_tag_of(op);
		if (!board) {
			console.error(op + ' has no tags.');
			process.exit(-1);
		}

		console.log('Dumping thread...');

		var base = joinPath(DUMP_DIR, op.toString());
		var authBase = joinPath(AUTH_DUMP_DIR, op.toString());
		var outputs = {
			auth: fs.createWriteStream(authBase + '.json'),
			json: fs.createWriteStream(base + '.json'),
			html: fs.createWriteStream(base + '.html'),
		};

		dump_thread(op, board, DUMP_IDENT, outputs, function (err) {
			if (err)
				throw err;

			var streams = [];
			for (var k in outputs)
				streams.push(outputs[k]);
			async.each(streams, close_stream, quit);
		});
	});

	function quit() {
		// crappy flush for stdout (can't close it)
		if (process.stdout.write(''))
			process.exit(0);
		else
			process.stdout.on('drain', function () {
				process.exit(0);
			});
	}
})();
