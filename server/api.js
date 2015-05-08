/*
 Read-only JSON API
 */

'use strict';

var _ = require('underscore'),
	caps = require('./caps'),
	config = require('../config'),
	db = require('../db'),
	express = require('express'),
	state = require('./state');

var app = express();
const JSONHeaders = {
	'Access-Control-Allow-Origin': '*',
	'Content-Type': 'application/json; charset=UTF-8',
	'Expires': 'Thu, 01 Jan 1970 00:00:00 GMT',
	'Cache-Control': 'no-cache, no-store'
};
var r = global.redis;
// On a different port for now. Will migrate everything to express some day
app.listen(config.API_PORT);

app.get(/api\/(post|thread)\/([0-9,]+)\/?/, function(req, res) {
	res.set(JSONHeaders);
	const nums = req.params[1].split(',');

	// If don't have access to even one board, return 404
	for (let i = 0, l = nums.length; i < l; i++) {
		let num = nums[i];
		if (invalid(req, config.BOARDS[isOP(num)
			? db.TAGS[num] : db.TAGS[db.OPs[num]]])
		)
			return res.sendStatus(404);
	}

	function respond(err, posts) {
		if (err)
			return res.send(err);
		// Posts not found for some reason
		if (!posts || posts.length === 0)
			return res.sendStatus(404);
		res.json(posts);
	}

	if (req.params[0] == 'post')
		getPosts(nums, respond);
	else
		getThreads(nums, Infinity, respond);
});

app.get(/\/api\/(catalog|board)\/([a-z0-9]+)\/?/, function(req, res) {
	res.set(JSONHeaders);
	const par = req.params;

	if (invalid(req, par[1]))
		return res.sendStatus(404);

	// Limit of replies to read
	let limit = par[0] == 'board' ? state.hot.THREADS_PER_PAGE : 0;

	// Read threads in reverse order from redis
	r.zrange(`tag:${db.tag_key(par[1])}:threads`, 0, -1, function(err, nums) {
		if (err)
			return res.send(err);
		if (!nums || nums.length === 0)
			return res.sendStatus(404);

		getThreads(nums.reverse(), limit, function(err, threads) {
			if (err)
				res.send(err);
			if (!threads || threads.length === 0)
				return res.sendStatus(404);
			// Arrays of arrays with only one element is uneeded complexity
			if (par[0] == 'catalog')
				threads = _.flatten(threads);
			res.json(threads);
		});
	});
});

// Expose client-side configuration
app.get(/\/api\/config/, function(req, res) {
	res.set(JSONHeaders);
	res.json({
		config: state.clientConfig,
		hot: state.clientHotConfig
	});
});

// Check board existanace and access rights
function invalid(req, board) {
	let forward = req.headers['x-forwarded-for'],
		ip = config.TRUST_X_FORWARDED_FOR && forward ? forward
			: req.connection.remoteAddress;
	return !caps.can_access_board({ip: ip}, board);
}

function isOP(num) {
	return db.TAGS[num] !== undefined;
}

function getPosts(nums, cb) {
	var posts = [],
		m = r.multi();
	// Read all of the posts
	for (let i = 0, l = nums.length; i < l; i++) {
		let num = nums[i];
		const key = (isOP(num) ? 'thread:' : 'post:') + num;
		// Posts the current post is linking to
		m.hgetall(key);
		m.hgetall(key + ':links');
	}
	m.exec(function(err, data) {
		if (err)
			return cb(err);
		for (let i = 0, l = data.length; i < l; i += 2) {
			let post = data[i];
			const links = data[i + 1];
			if (!post)
				continue;
			if (links)
				post.links = links;
			pruneData(post);
			posts.push(post);
		}
		// No posts retrieved
		if (posts.length === 0)
			return cb(null, null);
		cb(null, posts);
	});
}

function pruneData(data) {
	// Privacy
	delete data.ip;
	// Useless on the client
	delete data.hash;
	delete data.hctr;
	delete data.tags;
}

function getThreads(nums, replyLimit, cb) {
	let threads = [],
		m = r.multi();
	for (let i = 0, l = nums.length; i < l; i++) {
		let num = nums[i];
		// Return 404, if even one of the threads is not an OP
		if (!isOP(num))
			return cb(null, null);
		const key = 'thread:' + num;
		m.hgetall(key);
		m.hgetall(key + ':links');
		// Deleted posts are still present in the replies list
		// Don't need them to show up in the JSON
		m.lrange(key + ':dels', 0, -1);
		m.lrange(key + ':posts', 0, -1);
	}
	m.exec(function(err, data) {
		if (err)
			return cb(err);
		var op, links, dels, replies, allReplies = [];
		for (let i = 0, l = data.length; i < l; i += 4) {
			let op = data[i];
			const links = data[i + 1];
			const dels = data[i + 2];
			let replies = data[i + 3];
			if (!op)
				continue;
			pruneData(op);
			if (links)
				op.links = links;
			if (dels.length > 0)
			// Substract dels from replies
				replies = _.difference(replies, dels);
			op.replies = replies.length;
			// Only show the last n replies
			replies = _.last(replies, replyLimit);
			threads.push([op]);
			allReplies.push(replies);
		}

		allReplies = _.flatten(allReplies);
		// No replies ;_;
		if (allReplies.length === 0)
			return cb(null, threads);

		getPosts(allReplies, function(err, replies) {
			if (err)
				return cb(err);
			if (!replies)
				return cb(null, threads);
			// Ditribute replies among threads
			for (let i = 0, l = threads.length; i < l; i++) {
				let thread = threads[i];
				for (let o = 0, l = thread[0].length; o < l; o++) {
					thread.push(replies.shift());
				}
			}
			cb(null, threads);
		});
	});
}
