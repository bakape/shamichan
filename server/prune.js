/*
 Deletes threads that go past the last page of the board
 */

'use strict';

let _ = require('underscore'),
	async = require('async'),
	config = require('../config'),
    db = require('../db'),
	state = require('./state'),
    winston = require('winston');

let yaku;
function connect() {
	if (!yaku)
		yaku = new db.Yakusoku(null, db.UPKEEP_IDENT);
	return yaku.connect();
}

function yandere() {
	let m = connect().multi();
	const boards = config.BOARDS;
	for (let i = 0, l = boards.length; i < l; i++) {
		const key = `tag:${db.tag_key(boards[i])}:threads`;
		m.zrevrange(key, 0, -1);
	}
	m.exec(function(err, res) {
		if (err)
			winston.error(err);
		let toPrune = {};
		const pages = config.PAGES,
			perPage = state.hot.THREADS_PER_PAGE;
		for (let i = 0, l = res.length; i < l; i++) {
			let threads = res[i];
			// Board has no threads
			if (!threads.length)
				continue;

			const board = boards[i],
				// Threads that are over the last page
				over = _.rest(threads, pages[board] * perPage);
			if (!over.length)
				continue;
			for (let o = 0, len = over.length; o < len; o++) {
				toPrune[over[o]] = board;
			}
		}
		if (_.isEmpty(toPrune))
			return;
		// Done sequentially for performance reasons
		async.forEachOfSeries(toPrune, function(board, thread, cb) {
			yaku.purge_thread(thread, board, function(err) {
				if (err) {
					return winston.error('Thread purging error: '
						+ err.toString()
					);
				}
				winston.info('Purged thread: ' + thread);
				cb();
			});
		});
	})
}

setInterval(yandere, 60000);
yandere();
