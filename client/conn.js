(function () {

var socket, attempts, attemptTimer;

window.send = function (msg) {
	if (connSM.state != 'synced' && connSM.state != 'syncing')
		return;
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
	socket = window.new_socket(attempts);
	socket.onopen = connSM.feeder('open');
	socket.onclose = connSM.feeder('close');
	socket.onmessage = on_message;
}

window.new_socket = function (attempt) {
	return new SockJS(SOCKET_PATH);
};

connSM.act('conn, reconn + open -> syncing', function () {
	sync_status('Syncing...', false);
	sessionId = random_id();
	send([SYNCHRONIZE, sessionId, BOARD, syncs, BUMP, document.cookie]);
});

connSM.act('syncing + sync -> synced', function () {
	sync_status('Synced.', false);
	attemptTimer = setTimeout(function () {
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
	if (attempts++ < 10)
		setTimeout(connSM.feeder('retry'), 250 * Math.pow(2,attempts));
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
});

$(function () {
	_.defer(connSM.feeder('start'));
	$(window).focus(function () {
		setTimeout(connSM.feeder('retry'), 20);
	});
});

})();
