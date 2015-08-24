/*
Migrates a major semver 0 database to version 1
 */
'use strict';

const _ = require('underscore'),
	async  = require('async'),
	config = require('../config');

const redis = require('redis').createClient(config.REDIS_PORT);

async.waterfall(
	[
		function (next) {
			async.eachSeries(config.BOARDS, scanBoard, next);
		}, 
		function (next) {
			redis.set('dbVersion', 1, next);
		}
	],
	function (err) {
		if (err)
			throw error;
		console.log('Done');
		process.exit();
	}
);

function scanBoard(board, cb) {
	console.log(`Scanning /${board}/`);
	const tagKeys = [];
	async.waterfall([
		function (next) {
			redis.zrange(tagKey(board) + ':threads', 0, -1, next);
		}, 
		function (threads, next) {
			async.eachSeries(threads, scanThread.bind(null, board), next);
		},
		// Check for existance of all possible tagkeys
		function (next) {
			console.log('  renaming tag keys...');
			const suffixes = ['', ':threads', ':postctr', ':bumpctr'],
				m = redis.multi(),
				base = tagKey(board);
			for (let suffix of suffixes) {
				const key = base + suffix;
				tagKeys.push(key);
				m.exists(key);
			}
			m.exec(next);
		},
		// Rename tag keys
		function (res, next) {
			const m = redis.multi();
			for (let i = 0; i < res.length; i++) {
				if (!res[i])
					continue;
				const split = tagKeys[i].split(':');
				let newName = `board:${split[2]}`;
				if (split[3])
					newName += `:${split[3]}`;
				m.rename(tagKeys[i], newName);
			}
			m.exec(next);
		}
	], cb);
}

function scanThread(board, thread, cb) {
	console.log(`  thread ${thread}`);
	const keys = [],
		nums = [],
		key = 'thread:' + thread;
	let replyNums;
	async.waterfall([
		function (next) {
			redis.exists(key, next);
		},
		// We need to make sure the post key exists, or we would be creating
		// half-empty hashes
		function (exists, next) {
			if (!exists)
				// Remove from thread list and exit chain
				return redis.zrem(tagKey(board) + ':threads', thread, cb);
			keys.push(key);
			nums.push(thread);
			redis.lrange(key + ':posts', 0, -1, next);
		},
		function (posts, next) {
			const m = redis.multi();
			replyNums = posts;
			for (let post of posts) {
				m.exists(`post:${post}`);
			}
			m.exec(next);
		},
		function (exists, next) {
			const m = redis.multi();
			for (let num of replyNums) {
				if (exists) {
					keys.push(`post:${num}`);
					nums.push(num);
				}
				else
					// Remove from reply list
					m.lrem(key + ':posts', num);
			}
			m.exec(next);
		}, 
		function (res, next) {
			const m = redis.multi();
			for (let i = 0; i < keys.length; i++) {
				const key = keys[i];
				// Update hash attributes for all keys
				m.hmset(key, 'board', board, 'num', nums[i]);
				// Delete depricated dice properties. Can't be bothered to port
				// them.
				m.hdel(key, 'dice');
				// Get image properties, if any
				m.hget(key, 'src');
			}
			m.exec(next);
		},
		// Update image property spec
		function (res, next) {
			const m = redis.multi();
			for (let i = 0; i < res.length; i += 3) {
				const src = res[i + 2];
				if (!src)
					continue;
				m.hset(keys[i / 3], 'ext', _.last(src.split('.')));
			}
			m.exec(next);
		}
	], cb)
}

function tagKey(board) {
	return `tag:${board.length}:${board}`;
}
