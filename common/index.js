/*
 This file is used both by the server and client
 Keep that in mind, when making modifications
 */

'use strict';

// Websocket message codes and some other defines
const DEF = {
	INVALID: 0,
	INSERT_POST: 2,
	UPDATE_POST: 3,
	FINISH_POST: 4,
	// Legacy?
	CATCH_UP: 5,
	INSERT_IMAGE: 6,
	SPOILER_IMAGES: 7,
	DELETE_IMAGES: 8,
	DELETE_POSTS: 9,
	DELETE_THREAD: 10,
	LOCK_THREAD: 11,
	UNLOCK_THREAD: 12,
	REPORT_POST: 13,

	IMAGE_STATUS: 31,
	SYNCHRONIZE: 32,
	EXECUTE_JS: 33,
	UPDATE_BANNER: 35,
	ONLINE_COUNT: 37,
	HOT_INJECTION: 38,
	NOTIFICATION: 39,
	RADIO: 40,
	RESYNC: 41,

	MODEL_SET: 50,
	COLLECTION_RESET: 55,
	COLLECTION_ADD: 56,
	SUBSCRIBE: 60,
	UNSUBSCRIBE: 61,
	GET_TIME: 62,

	INPUT_ROOM: 20,
	MAX_POST_LINES: 30,
	MAX_POST_CHARS: 2000,
	WORD_LENGTH_LIMIT: 300,

	S_NORMAL: 0,
	S_BOL: 1,
	S_QUOTE: 2,
	S_SPOIL: 3
};

// Export everything in one big object
module.exports = exports = require('./util');
require('underscore').extend(exports, DEF);
exports.OneeSama = require('./oneesama');
