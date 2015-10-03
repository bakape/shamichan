/*
 This file is used both by the server and client
 Keep that in mind, when making modifications
 */

// Websocket message codes and some other defines
const DEF = {
	// Pub/Sub
	INVALID: 0,
	BACKLINK: 1,
	INSERT_POST: 2,
	UPDATE_POST: 3,
	FINISH_POST: 4,
	INSERT_IMAGE: 6,

	// Pub/sub moderation
	SPOILER_IMAGES: 7,
	DELETE_IMAGES: 8,
	DELETE_POSTS: 9,
	LOCK_THREAD: 10,
	UNLOCK_THREAD: 11,
	BAN: 12,
	REPORT_POST: 13,

	// Miscellaneous
	IMAGE_STATUS: 31,
	SYNCHRONIZE: 32,
	EXECUTE_JS: 33,
	// Unsubscribe from server-side redis listeners
	DESYNC: 34,
	UPDATE_BANNER: 35,
	GET_TIME: 36,
	ONLINE_COUNT: 37,
	HOT_INJECTION: 38,
	NOTIFICATION: 39,
	RADIO: 40,
	RESYNC: 41,

	// Non-pub/sub Moderation
	MOD_LOG: 51,
	ADMIN_PANEL: 52,
	UNBAN: 53,

	// Various constants
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
