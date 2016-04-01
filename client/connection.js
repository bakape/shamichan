/*
 * Websocket controller and connection notifier
 */

const main = require('./main'),
	{_, common, config, connSM, state} = main,
	lang = main.lang.sync;

let socket, attempts, attemptTimer;

function send(msg) {
	// need deferral or reporting on these lost messages...
	if (connSM.state != 'synced' && connSM.state != 'syncing')
		return;
	if (socket.readyState != SockJS.OPEN) {
		if (console)
			console.warn("Attempting to send while socket closed");
		return;
	}

	msg = JSON.stringify(msg);
	if (config.DEBUG)
		console.log('<', msg);
	socket.send(msg);
}
main.reply('send', send);

function on_message(e) {
	if (config.DEBUG)
		console.log('>', e.data);

	const msg = JSON.parse(e.data)[0],
		op = msg.shift(),
		type = msg.shift(),
		isPubSub = common.is_pubsub(type);

	if (isPubSub && connSM.state === 'locked')
		return;

	// Some handlers are optional and/or dynamic. Ignore them silently.
	const handler = main.dispatcher[type];
	if (!handler)
		return;
	if (isPubSub && op in state.syncs)
		state.syncs[op]++;
	main.follow(() => handler(msg, op));
}

const sync = document.getElementById('sync');
function sync_status(msg) {
	sync.textContent = msg;
}

connSM.act('load + start -> conn', function () {
	sync_status(lang.connecting);
	attempts = 0;
	connect();
});

function connect() {
	if (socket) {
		socket.onclose = null;
		socket.onmessage = null;
	}
	if (window.location.protocol == 'file:') {
		console.log("Page downloaded locally; refusing to sync.");
		return;
	}
	socket = new_socket();
	socket.onopen = connSM.feeder('open');
	socket.onclose = connSM.feeder('close');
	socket.onmessage = on_message;
	if (config.DEBUG)
		window.socket = socket;
}

function new_socket() {
	const transports = ['xdr-streaming', 'xhr-streaming', 'iframe-eventsource',
		'iframe-htmlfile', 'xdr-polling', 'xhr-polling', 'iframe-xhr-polling',
		'jsonp-polling'];
	if (config.USE_WEBSOCKETS)
		transports.unshift('websocket');
	return new SockJS(config.SOCKET_URL || config.SOCKET_PATH, null, {
		transports
	});
}

connSM.act('conn, reconn + open -> syncing', () => {
	sync_status(lang.syncing);
	const connID = common.random_id(),
		{page} = state;
	page.set('connID', connID);
	send([common.SYNCHRONIZE, connID, page.get('board'), state.syncs,
		page.get('live'), document.cookie]);
});

connSM.act('syncing + sync -> synced', function () {
	sync_status(lang.synced);
	attemptTimer = setTimeout(function () {
		attemptTimer = 0;
		reset_attempts();
	}, 10000);
});

function reset_attempts() {
	if (attemptTimer) {
		clearTimeout(attemptTimer);
		attemptTimer = 0;
	}
	attempts = 0;
}

// Prevent pub/sub mesages from being handled
connSM.act('synced, syncing + lock -> locked');
connSM.act('locked + unlock -> synced');
main.reply('connection:lock', () => send([common.DESYNC]), connSM.feed('lock'));
main.reply('connection:unlock', msg => send(msg), connSM.feed('unlock'));

connSM.act('* + close -> dropped',  error => {
	if (socket) {
		socket.onclose = null;
		socket.onmessage = null;
	}
	if (config.DEBUG)
		console.error('E:', error);
	if (attemptTimer) {
		clearTimeout(attemptTimer);
		attemptTimer = 0;
	}
	sync_status(lang.dropped);

	// Wait maxes out at ~1min
	const wait = 500 * Math.pow(1.5, Math.min(Math.floor(++attempts / 2), 12));
	setTimeout(connSM.feeder('retry'), wait);
});

connSM.act('dropped + retry -> reconn', function () {
	connect();
	/* Don't show this immediately so we don't thrash on network loss */
	setTimeout(function () {
		if (connSM.state == 'reconn')
			sync_status(lang.reconnecting);
	}, 100);
});

connSM.act('* + invalid, desynced + close -> desynced', msg => {
	msg = (msg && msg[0]) ? 'Out of sync: ' + msg[0] : 'Out of sync';
	sync_status(msg);
	if (attemptTimer) {
		clearTimeout(attemptTimer);
		attemptTimer = 0;
	}
	socket.onclose = null;
	socket.onmessage = null;
	socket.close();
	socket = null;
	if (config.DEBUG)
		window.socket = null;
});

function window_focused() {
	switch (connSM.state) {
		case 'desynced':
			return;
		// might have just been suspended;
		// try to get our FSM up to date if possible
		case 'synced':
		case 'syncing':
		case 'conn':
			const rs = socket.readyState;
			if (rs != SockJS.OPEN && rs != SockJS.CONNECTING) {
				connSM.feed('close');
				return;
			}
			else if (navigator.onLine === false) {
				connSM.feed('close');
				return;
			}
			break;
	}
	connSM.feed('retry');
}

// Connect to server
connSM.feed('start');

// Check for connectivity each time tab visibility changes to visible
// A bit of an overhead, but should prevent unregistered disconnects,
// especially on mobile.
document.addEventListener('visibilitychange', e => {
	if (e.target.hidden)
		return;
	setTimeout(window_focused, 20);
});
window.addEventListener('online', () => reset_attempts(), connSM.feed('retry'));
window.addEventListener('offline', connSM.feeder('close'));
