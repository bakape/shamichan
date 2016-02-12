/*
 Deletes threads that go past the last page of the board
 */

let _ = require('underscore'),
	async = require('async'),
	config = require('../config'),
    db = require('../db'),
	state = require('./state'),
    winston = require('winston');

const yaku = new db.Yakusoku(null, db.UPKEEP_IDENT),
	redis = global.redis;

function yandere() {
	const m = redis.multi();
	for (let board of config.BOARDS) {
		m.zrevrange(`board:${board}:threads`, 0, -1);
	}
	m.exec(function (err, res) {
		if (err)
			winston.error(err);
		const toPrune = {};
		for (let i = 0; i < res.length; i++) {
			const threads = res[i];
			
			// Board has no threads
			if (!threads.length)
				continue;

			const board = config.BOARDS[i],
				// Threads that are over the last page
				over = _.rest(threads, 
					config.PAGES[board] * state.hot.THREADS_PER_PAGE);
			for (let thread of over) {
				toPrune[thread] = board;
			}
		}
		if (_.isEmpty(toPrune))
			return;
		// Done sequentially for performance reasons
		async.forEachOfSeries(toPrune, function(board, thread, cb) {
			yaku.purge_thread(thread, board, function (err) {
				if (err)
					winston.error('Thread purging error:', err);
				else
					winston.info('Purged thread: ' + thread);
				cb();
			});
		});
	})
}

setInterval(yandere, 60000);
yandere();
