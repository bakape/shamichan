/*
Various utility functions
 */

const config = require('../config'),
	child_process = require('child_process'),
    fs = require('fs'),
    util = require('util');

/* Non-wizard-friendly error message */
function Muggle(message, reason) {
	// Enable passing an array instead of 2 arguments
	if (message instanceof Array) {
		reason = message[1];
		message = message[0];
	}
	if (!(this instanceof Muggle))
		return new Muggle(message, reason);
	Error.call(this, message);
	Error.captureStackTrace(this, this.constructor);
	this.message = message;
	this.reason = reason;
}
util.inherits(Muggle, Error);
exports.Muggle = Muggle;

Muggle.prototype.most_precise_error_message = function () {
	var deepest = this.message;
	var muggle = this;
	var sanity = 10;
	while (muggle.reason && muggle.reason instanceof Muggle) {
		muggle = muggle.reason;
		if (muggle.message && typeof muggle.message == 'string')
			deepest = muggle.message;
		if (--sanity <= 0)
			break;
	}
	return deepest;
};

Muggle.prototype.deepest_reason = function () {
	if (this.reason && this.reason instanceof Muggle)
		return this.reason.deepest_reason();
	return this;
};

function move (src, dest, callback) {
	child_process.execFile('/bin/mv', ['--', src, dest],
				function (err, stdout, stderr) {
		if (err)
			callback(Muggle("Couldn't move file into place.",
					stderr || err));
		else
			callback(null);
	});
}
exports.move = move;

function movex (src, dest, callback) {
	child_process.execFile('/bin/mv', ['-n', '--', src, dest],
				function (err, stdout, stderr) {
		if (err)
			callback(Muggle("Couldn't move file into place.",
					stderr || err));
		else
			callback(null);
	});
}
exports.movex = movex;

function cpx (src, dest, callback) {
	child_process.execFile('/bin/cp', ['-n', '--', src, dest],
				function (err, stdout, stderr) {
		if (err)
			callback(Muggle("Couldn't copy file into place.",
					stderr || err));
		else
			callback(null);
	});
}
exports.cpx = cpx;

function checked_mkdir (dir, cb) {
	fs.mkdir(dir, function (err) {
		cb(err && err.code == 'EEXIST' ? null : err);
	});
}
exports.checked_mkdir = checked_mkdir;

// Get binary absolute path
function which(name, callback) {
	child_process.exec('which ' + name, function (err, stdout, stderr) {
		if (err)
			throw err;
		callback(stdout.trim());
	});
}
exports.which = which;

// Veryfies a client's setting is compatible with the server's. Otherwise
// returns default.
function resolveConfig(server, client, def) {
	if (~server.indexOf(client))
		return client;
	return def;
}
exports.resolveConfig = resolveConfig;
