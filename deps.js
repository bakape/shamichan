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
		'server',
		'util',
		'db.js'
	],
	state: [
		'tmpl',
		'config/hot.js'
	],
	client: [
		'client'
	],
	mod: [
		'client/mod'
	],
	get legacy() {
		return this.client;
	},
	css: [
		'less'
	]
};
