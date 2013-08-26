var async = require('async'),
    config = require('./config'),
    crypto = require('crypto'),
    etc = require('./etc'),
    fs = require('fs'),
    make_client = require('./make_client').make_maybe_minified,
    pathJoin = require('path').join,
    stream = require('stream'),
    tmp_file = require('tmp').file,
    util = require('util');

const PUBLIC_JS = pathJoin('www', 'js');

function HashingStream(out) {
	stream.Writable.call(this);

	this._hash = crypto.createHash('MD5');
	this._outStream = out;
}
util.inherits(HashingStream, stream.Writable);

HashingStream.prototype._write = function (chunk, encoding, cb) {
	this._hash.update(chunk);
	this._outStream.write(chunk, encoding, cb);
};

HashingStream.prototype.end = function (cb) {
	if (arguments.length > 1)
		throw new Error("TODO multi-arg HashingStream.end");
	var self = this;
	stream.Writable.prototype.end.call(this, function () {
		self._outStream.end(function () {
			if (cb)
				cb();
		});
	});
};

function end_and_move_js(stream, dir, prefix, cb) {
	stream.end(function () {
		var fnm;
		if (config.DEBUG) {
			fnm = prefix + '-debug.js';
		}
		else {
			var hash = stream._hash.digest('hex').slice(0, 10);
			fnm = prefix + '-' + hash + '.min.js';
		}
		var tmp = stream._tmpFilename;
		etc.move(tmp, pathJoin(dir, fnm), function (err) {
			if (err)
				return cb(err);
			cb(null, fnm);
		});
	});
};


function make_hashing_stream(cb) {
	// ideally the stream would be returned immediately and handle
	// this step internally...
	tmp_file({dir: '.build', postfix: '.gen.js'}, function (err, tmp, fd) {
		if (err)
			return cb(err);
		var out = fs.createWriteStream(null, {fd: fd});
		out.once('error', cb);

		if (config.DEBUG) {
			out._tmpFilename = tmp;
			cb(null, out);
		}
		else {
			var stream = new HashingStream(out);
			stream._tmpFilename = tmp;
			cb(null, stream);
		}
	});
}

function build_vendor_js(cb) {
	var deps = require('./deps');
	make_hashing_stream(function (err, stream) {
		if (err)
			return cb(err);
		async.eachSeries(deps.VENDOR_DEPS, function (file, cb) {
			fs.readFile(file, function (err, buf) {
				if (err)
					return cb(err);
				stream.write(buf, cb);
			});
		}, function (err) {
			if (err)
				return cb(err);
			end_and_move_js(stream, PUBLIC_JS, 'vendor', cb);
		});
	});
}

function build_client_js(cb) {
	var deps = require('./deps');
	make_hashing_stream(function (err, stream) {
		if (err)
			return cb(err);
		make_client(deps.CLIENT_DEPS, stream, function (err) {
			if (err)
				return cb(err);
			end_and_move_js(stream, PUBLIC_JS, 'client', cb);
		});
	});
}

function build_mod_client_js(cb) {
	var deps = require('./deps');
	make_hashing_stream(function (err, stream) {
		if (err)
			return cb(err);
		make_client(deps.MOD_CLIENT_DEPS, stream, function (err) {
			if (err)
				return cb(err);
			end_and_move_js(stream, 'state', 'mod', cb);
		});
	});
}

function commit_assets(metadata, cb) {
	tmp_file({dir: '.build', postfix: '.json'}, function (err, tmp, fd) {
		if (err)
			return cb(err);
		var stream = fs.createWriteStream(null, {fd: fd});
		stream.once('error', cb);
		stream.end(JSON.stringify(metadata) + '\n', function () {
			etc.move(tmp, pathJoin('state', 'scripts.json'), cb);
		});
	});
}

function rebuild(cb) {
	etc.checked_mkdir('state', function (err) {
	etc.checked_mkdir('.build', function (err) {
		if (err) return cb(err);
		async.parallel({
			vendor: build_vendor_js,
			client: build_client_js,
			mod: build_mod_client_js,
		}, function (err, hashes) {
			if (err)
				return cb(err);
			commit_assets(hashes, cb);
		});
	});
	});
}
exports.rebuild = rebuild;

exports.refresh_deps = function () {
	delete require.cache[pathJoin(__dirname, 'deps.js')];
};

if (require.main === module) {
	rebuild(function (err) {
		if (err) throw err;
	});
}
