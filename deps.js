/*
List of directories, changes in which should trigger appropriate
rebuilds/restarts in builder.js. This file no longer has any other function.

There is a bit of overhead, because we watch entired directories recursicely,
but who cares. Writing in each file individually is a pain.
 */

exports.server = [
	'admin',
	'common',
	'config',
	'imager',
	'lang',
	'server',
	'util',
	'db.js'
];

exports.state = [
	'tmpl',
	'config/hot.js'
];

// TEMP: Still used for building
exports.mod = [
	'admin/index.js',
	'admin/client.js'
];

exports.client = [
	'gulpfile.js',
	'common',
	'client'
];

exports.css = ['less'];
