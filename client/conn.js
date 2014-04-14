(function () {

var socket, attempts, attemptTimer;

window.send = function (msg) {
	// need deferral or reporting on these lost messages...
	if (connSM.state != 'synced' && connSM.state != 'syncing')
		return;
	if (socket.readyState != SockJS.OPEN) {
		if (console)
			console.warn("Attempting to send while socket closed");
		return;
	}

	msg = JSON.stringify(msg);
	if (DEBUG)
		console.log('<', msg);
	socket.send(msg);
};

function on_message(e) {
	if (DEBUG)
		console.log('>', e.data);
	var msgs = JSON.parse(e.data);

	with_dom(function () {

	for (var i = 0; i < msgs.length; i++) {
		var msg = msgs[i];
		var op = msg.shift();
		var type = msg.shift();
		if (is_pubsub(type) && op in syncs)
			syncs[op]++;
		dispatcher[type](msg, op);
	}

	});
}

function sync_status(msg, hover) {
	$('#sync').text(msg).attr('class', hover ? 'error' : '');
}

connSM.act('load + start -> conn', function () {
	sync_status('Connecting...', false);
	attempts = 0;
	connect();
});

function connect() {
	if (window.location.protocol == 'file:') {
		console.log("Page downloaded locally; refusing to sync.");
		return;
	}
	socket = window.new_socket(attempts);
	socket.onopen = connSM.feeder('open');
	socket.onclose = connSM.feeder('close');
	socket.onmessage = on_message;
	if (DEBUG)
		window.socket = socket;
}

window.new_socket = function (attempt) {
	return new SockJS(SOCKET_PATH);
};

connSM.act('conn, reconn + open -> syncing', function () {
	sync_status('Syncing...', false);
	CONN_ID = random_id();
	send([SYNCHRONIZE, CONN_ID, BOARD, syncs, BUMP, document.cookie]);
});

connSM.act('syncing + sync -> synced', function () {
	sync_status('Synced.', false);
	attemptTimer = setTimeout(function () {
		attemptTimer = 0;
		attempts = 0;
	}, 10000);
});

connSM.act('* + close -> dropped', function (e) {
	if (DEBUG)
		console.error('E:', e);
	if (attemptTimer) {
		clearTimeout(attemptTimer);
		attemptTimer = 0;
	}
	sync_status('Dropped.', true);

	attempts++;
	var n = Math.min(Math.floor(attempts/2), 12);
	var wait = 500 * Math.pow(1.5, n);
	// wait maxes out at ~1min
	setTimeout(connSM.feeder('retry'), wait);
});

connSM.act('dropped + retry -> reconn', function () {
	connect();
	/* Don't show this immediately so we don't thrash on network loss */
	setTimeout(function () {
		if (connSM.state == 'reconn')
			sync_status('Reconnecting...', true);
	}, 100);
});

connSM.act('* + invalid, desynced + close -> desynced', function (msg) {
	msg = (msg && msg[0]) ? 'Out of sync: ' + msg[0] : 'Out of sync.';
	sync_status(msg, true);
	if (attemptTimer) {
		clearTimeout(attemptTimer);
		attemptTimer = 0;
	}
	delete socket.onclose;
	socket.close();
	socket = null;
	if (DEBUG)
		window.socket = null;
});

function window_focused() {
	var s = connSM.state;
	if (s == 'desynced')
		return;
	// might have just been suspended;
	// try to get our SM up to date if possible
	if (s == 'synced' || s == 'syncing' || s == 'conn') {
		var rs = socket.readyState;
		if (rs != SockJS.OPEN && rs != SockJS.CONNECTING) {
			connSM.feed('close');
			return;
		}
		else if (navigator.onLine === false) {
			connSM.feed('close');
			return;
		}
	}
	connSM.feed('retry');
}

$(function () {
	_.defer(connSM.feeder('start'));
	$(window).focus(function () {
		setTimeout(window_focused, 20);
	});
	window.addEventListener('online', connSM.feeder('retry'));
	window.addEventListener('offline', connSM.feeder('close'));
});

})();
