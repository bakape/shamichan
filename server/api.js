var _ = require('underscore'),
	caps = require('./caps'),
	config = require('../config'),
	db = require('../db'),
	express = require('express'),
	state = require('./state');

var app = express();
var JSONHeaders = {
	'Access-Control-Allow-Origin': '*',
	'Content-Type': 'application/json; charset=UTF-8',
	'Expires': 'Thu, 01 Jan 1970 00:00:00 GMT',
	'Cache-Control': 'no-cache, no-store'
};
var r = global.redis;
// On a different port for now. Will migrate everything to express some day
app.listen(config.API_PORT);

app.get(/api\/(post|thread)\/([0-9]+)\/?/, function(req, res){
	res.set(JSONHeaders);
	var par = req.params,
		isOP = db.TAGS[par[1]] !== undefined,
		board = config.BOARDS[(isOP ? db.TAGS[par[1]] : db.TAGS[db.OPs[par[1]]])];

	if (invalid(req, board))
		return res.sendStatus(404);

	function respond(err, posts) {
		if (err)
			return res.send(err);
		// Posts not found for some reason
		if (!posts ||posts.length === 0)
			return res.sendStatus(404);
		// Threads and posts come as an array inside and array, for interoperability with
		// catalog and board requests
		res.json(posts[0]);
	}

	if (par[0] == 'post')
		getPosts([par[1]], isOP, respond);
	else if (isOP)
		getThreads([par[1]], Infinity, respond);
	else
		res.sendStatus(404);
});

app.get(/\/api\/(catalog|board)\/([a-z0-9]+)\/?/, function(req, res){
	res.set(JSONHeaders);
	var par = req.params;;

	if (invalid(req, par[1]))
		return res.sendStatus(404);

	// Limit of replies to read
	var limit = par[0] == 'board' ? state.hot.THREADS_PER_PAGE : 0;

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

// Check board existanace and access rights
function invalid(req, board){
	var forward = req.headers['x-forwarded-for'],
		ip = config.TRUST_X_FORWARDED_FOR && forward ? forward : req.connection.remoteAddress;
	if (!caps.can_access_board({ip: ip}, board))
		return true;
	return false;
};

function getPosts(nums, isOP, cb) {
		var posts = [],
			m = r.multi(),
			keyHeader = isOP ? 'thread:' : 'post:',
			key, links;

		// Read all of the posts
		for (var num of nums) {
			key = keyHeader + num;
			// Posts the current post is linking to
			links = key + ':links';
			m.hgetall(key);
			m.hgetall(links);
		}
		m.exec(function(err, data){
			if (err)
				return cb(err);
			var post, links;
			for (var i = 0; i < data.length; i += 2) {
				post = data[i];
				links = data[i + 1];
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

function pruneData(data){
	// Privacy
	delete data.ip;
	// Useless on the client
	delete data.hash;
	delete data.hctr;
	delete data.tags;
}

function getThreads(nums, replyLimit, cb) {
	var threads = [], m = r.multi(), key;
	for (var num of nums) {
		key = 'thread:' + num;
		m.hgetall(key);
		m.hgetall(key + ':links');
		// Deleted posts are still present in the replies list
		// Don't need them to show up in the JSON
		m.lrange(key + ':dels', 0, -1);
		m.lrange(key + ':posts', 0, -1);
	}
	m.exec(function(err, data){
		if (err)
			return cb(err);
		var op, links, dels, replies, allReplies = [];
		for (var i = 0; i < data.length; i += 4) {
			op = data[i];
			links = data[i + 1];
			dels = data[i + 2];
			replies = data[i + 3];
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

		getPosts(allReplies, false, function(err, replies){
			if (err)
				return cb(err);
			if (!replies)
				return cb(null, threads);
			// Ditribute replies among threads
			for (var i = 0; i < threads.length; i++) {
				for (var o = 0; o < threads[i][0].replies; o++) {
					threads[i].push(replies.shift());
				}
			}
			cb(null, threads);
		});
	});
}
