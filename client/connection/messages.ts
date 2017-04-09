// Message types of the WebSocket communication protocol
export const enum message {
	invalid,

	// 1 - 29 modify post model state
	insertThread,
	insertPost,
	append,
	backspace,
	splice,
	closePost,
	backlink,
	insertImage,
	spoiler,
	deletePost,
	banned,

	// >= 30 are miscellaneous and do not write to post models
	synchronise = 30,
	reclaim,

	// Send new post ID to client
	postID,

	// Concatenation of multiple websocket messages to reduce transport overhead
	concat,

	// Invokes no operation on the server. Used to test the client's connection
	// in situations, when you can't be certain the client is still connected.
	NOOP,

	// Transmit current synced IP count to client
	syncCount,

	// Send current server Unix time to client
	serverTime,

	// Redirect the client to a specific board
	redirect,

	// Send a notification to a client
	notification,
}

export type MessageHandler = (msg: {}) => void

// Websocket message handlers. Each handler responds to its distinct message
// type.
export const handlers: { [type: number]: MessageHandler } = {}
