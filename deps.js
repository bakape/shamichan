/*
List of directories, changes in which should trigger appropriate
rebuilds/restarts in builder.js. This file no longer has any other function.

There is a bit of overhead, because we watch entired directories recursicely,
but who cares. Writing in each file individually is a pain.
 */
module.exports = {
	server: [
		'admin',
		'config',
		'imager',
		'lang',
		'server',
		'util',
		'db.js'
	],
	state: [
		'tmpl',
		'config/hot.js'
	],
	// TEMP: Still used for building
	mod: [
		'admin/index.js',
		'admin/client.js'
	],
	client: [
		'client'
	],
	css: [
		'less'
	]
};
