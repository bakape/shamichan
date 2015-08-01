/*
 Read-only JSON API
 */

let caps = require('../caps'),
	config = require('../../config'),
	db = require('../../db'),
	express = require('express'),
	state = require('../state'),
	util = require('./util'),
	winston = require('winston');

let app = module.exports = express();

// Use a subapp instead of a router, so we can have separate settings. The JSON
// responses aren't piped, like the HTML, so we can use the inbuilt hashing
// eTagger here
app.enable('strict routing');
app.use(function(req, res, next) {
	res.set({
		'Expires': 'Thu, 01 Jan 1970 00:00:00 GMT',
		'Cache-Control': 'no-cache, no-store',
		'Access-Control-Allow-Origin': '*',
		'Content-Type': 'application/json; charset=UTF-8'
	});
	next();
});

app.get(/^\/post\/(\d+)$/, function(req, res) {
	new db.Reader(req.ident)
		.singlePost(req.params[0], req.ident, function(post) {
			if (!post)
				return res.sendStatus(404);
			res.json(post);
		});
});

app.get(/^\/thread\/(\d+)$/, function(req, res) {
	const num = req.params[0],
		info = db.postInfo(num);
	if (!info.isOP || !caps.can_access_board(req.ident, info.board))
		return res.sendStatus(404);
	let reader = new db.Reader(req.ident),
		thread = [];
	reader.get_thread(info.board, num, {
		abbrev: req.query.last
	});
	reader.once('nomatch', function() {
		res.sendStatus(404);
	});
	reader.once('error', function(err) {
		winston.error(`thread ${num}:`, err);
		res.sendStatus(404);
	});
	reader.once('thread', function(post) {
		thread.push(post);
	});
	reader.on('post', function(post) {
		thread.push(post);
	});
	reader.once('end', function() {
		res.json(thread);
	})
});

// Array of a board's threads in order
app.get(/^\/board\/(\w+)$/, util.boardAccess, function(req, res) {
	const key = `tag:${db.tag_key(req.board)}:threads`;
	global.redis.zrevrange(key, 0, -1, function(err, threads) {
		if (err)
			return res.sendStatus(404);
		res.json(threads);
	})
});

app.get('/config', function(req, res) {
	res.json({
		config: state.clientConfig,
		hot: state.clientHotConfig
	});
});
